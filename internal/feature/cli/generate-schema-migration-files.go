package feature_cli

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"

	"ariga.io/atlas/sql/migrate"
	_ "ariga.io/atlas/sql/postgres" // Import Atlas PostgreSQL driver
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	_ "github.com/lib/pq" // Import PostgreSQL driver
)

// this has to be in separate file becuase of the imports
func migrateSchema(ctx context.Context) ([]string, []string) {
	// Convert GORM DSN to Atlas PostgreSQL URL format
	sourceDBUrl := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Default().Database.User,
		cfg.Default().Database.Password,
		cfg.Default().Database.Host,
		cfg.Default().Database.Port,
		cfg.Default().Database.Name,
		cfg.Default().Database.SSLMode)

	//setup clients
	sourceClient, err := sqlclient.Open(ctx, sourceDBUrl)
	if err != nil {
		panic(err.Error())
	}
	defer sourceClient.Close()

	targetDBUrl := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Default().Database.User,
		cfg.Default().Database.Password,
		cfg.Default().Database.Host,
		cfg.Default().Database.Port,
		cfg.Default().Database.MigrationName,
		cfg.Default().Database.SSLMode)

	targetClient, err := sqlclient.Open(ctx, targetDBUrl)
	if err != nil {
		panic(err.Error())
	}
	defer targetClient.Close()
	// Inspect the complete database realm so every schema is included.
	sourceSchema, err := sourceClient.InspectRealm(ctx, nil)
	if err != nil {
		panic(err.Error())
	}
	targetSchema, err := targetClient.InspectRealm(ctx, nil)
	if err != nil {
		panic(err.Error())
	}
	// Generate UP migration (from target to source)
	upChanges, err := sourceClient.RealmDiff(targetSchema, sourceSchema)
	if err != nil {
		panic(err.Error())
	}
	if len(upChanges) == 0 {
		log.Println("no changes in database schema")
		return nil, nil
	}
	// Generate UP and DOWN statements
	ups := []string{}
	downs := []string{}
	planOpt := func(opts *migrate.PlanOptions) {
		// Leave SchemaQualifier nil so realm changes such as AddSchema are allowed.
		opts.Indent = "  "
	}
	planApplier, ok := sourceClient.Driver.(migrate.PlanApplier)
	if !ok {
		panic("Unable to generate SQL - driver does not support PlanChanges")
	}
	upPlan, err := planApplier.PlanChanges(ctx, "migrate_schema_up", upChanges, planOpt)
	if err != nil {
		panic("Failed to generate UP migration plan: " + err.Error())
	}
	for _, stmt := range upPlan.Changes {
		// Skip schema_migrations and spatial_ref_sys table changes
		if strings.Contains(strings.ToLower(stmt.Cmd), "schema_migrations") ||
			strings.Contains(strings.ToLower(stmt.Cmd), "spatial_ref_sys") {
			continue
		}

		cmd := stmt.Cmd
		if addTable, ok := stmt.Source.(*schema.AddTable); ok {
			isUnlogged, err := checkTableIsUnlogged(ctx, sourceClient, addTable.T.Schema.Name, addTable.T.Name)
			if err != nil {
				panic("check unlogged error: " + err.Error())
			}
			if isUnlogged {
				cmd = strings.Replace(cmd, "CREATE TABLE", "CREATE UNLOGGED TABLE", 1)
			}
		}

		// Skip any auto_increment change
		var shouldSkip bool
		if modifyTable, ok := stmt.Source.(*schema.ModifyTable); ok {
			for _, change := range modifyTable.Changes {
				if modifyAttr, ok := change.(*schema.ModifyAttr); ok {
					if strings.Contains(fmt.Sprintf("%T", modifyAttr.From), "AutoIncrement") {
						shouldSkip = true
						break
					}
				}
			}
		}
		if shouldSkip {
			continue
		}
		ups = append(ups, normalizeGeneratedSchemaSQL(cmd))
		// down
		reverseStmts, err := stmt.ReverseStmts()
		if err != nil {
			panic(err.Error())
		}
		for i := range reverseStmts {
			reverseStmts[i] = normalizeGeneratedSchemaSQL(reverseStmts[i])
		}
		downs = append(downs, reverseStmts...)
	}
	// now all table schemas are in place, add in extensions and triggers
	// Prepend extensions at the beginning (before table creation)
	extensionsToAdd := []string{}

	// Check if postgis extension exists and add it if it doesn't
	postgisExtensionExists, err := checkExtensionExists(ctx, targetClient, "postgis")
	if err != nil {
		panic("Failed to check postgis extension existence: " + err.Error())
	}
	if !postgisExtensionExists {
		extensionsToAdd = append(extensionsToAdd, createPostgisExtension)
	}

	// Check if vector extension exists and add it if it doesn't
	vectorExtensionExists, err := checkExtensionExists(ctx, targetClient, "vector")
	if err != nil {
		panic("Failed to check vector extension existence: " + err.Error())
	}
	if !vectorExtensionExists {
		extensionsToAdd = append(extensionsToAdd, createVectorExtension)
	}

	// check if citext extension exists and add it if it doesn't
	citextExtensionExists, err := checkExtensionExists(ctx, targetClient, "citext")
	if err != nil {
		panic("Failed to check citext extension existence: " + err.Error())
	}
	if !citextExtensionExists {
		extensionsToAdd = append(extensionsToAdd, createCITextExtension)
	}

	// Prepend extensions to the beginning of ups array
	if len(extensionsToAdd) > 0 {
		ups = append(extensionsToAdd, ups...)
	}

	// Check if trigger function exists and add it if it doesn't
	functionExists, err := checkFunctionExists(ctx, targetClient, "update_last_update_column")
	if err != nil {
		panic("Failed to check trigger function existence: " + err.Error())
	}
	if !functionExists {
		ups = append(ups, updateLastUpdateFunction)
	}
	// Get all tables and create triggers for those that don't have them
	sourceTables, err := getAllTables(ctx, sourceClient)
	if err != nil {
		panic("Failed to get tables from source: " + err.Error())
	}
	for _, tableName := range sourceTables {
		triggerName := tableName[strings.LastIndex(tableName, ".")+1:] + "_set_last_update"
		triggerExists, err := checkTableTriggerExists(ctx, targetClient, tableName, triggerName)
		if err != nil {
			panic("Failed to check trigger existence for table " + tableName + ": " + err.Error())
		}
		if !triggerExists {
			up, down := getTableTriggerSQL(tableName, triggerName)
			ups = append(ups, up)
			downs = append(downs, down)
		}
	}
	return ups, downs
}

// constants
const updateLastUpdateFunction = `CREATE OR REPLACE FUNCTION update_last_update_column()
RETURNS TRIGGER AS $$
BEGIN
    -- Only operate on UPDATEs
    IF TG_OP = 'UPDATE' THEN
        -- Only consider when the row data actually changed
        IF (NEW.* IS DISTINCT FROM OLD.*) THEN
            -- If caller did not explicitly set last_updated_at (NULL)
            -- or left it equal to the old value, then set it to now().
            -- If caller set a different last_updated_at, respect that value.
            IF NEW.last_updated_at IS NULL OR NEW.last_updated_at = OLD.last_updated_at THEN
                NEW.last_updated_at = now();
            END IF;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`
const createPostgisExtension = `CREATE EXTENSION IF NOT EXISTS postgis`
const createVectorExtension = `CREATE EXTENSION IF NOT EXISTS vector`
const createCITextExtension = `CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public`

// utilities
var schemaQualifiedCITextPattern = regexp.MustCompile(`(?i)(?:"[^"]+"|[a-z_][a-z0-9_]*)\.citext\b`)

func normalizeGeneratedSchemaSQL(sql string) string {
	return schemaQualifiedCITextPattern.ReplaceAllString(sql, "public.citext")
}

func checkExtensionExists(ctx context.Context, client *sqlclient.Client, extensionName string) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(
		SELECT 1 FROM pg_extension 
		WHERE extname = '%s'
	)`, extensionName)
	rows, err := client.QueryContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var exists bool
	if rows.Next() {
		err = rows.Scan(&exists)
		if err != nil {
			return false, err
		}
	}
	return exists, nil
}

func checkFunctionExists(ctx context.Context, client *sqlclient.Client, functionName string) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(
		SELECT 1 FROM pg_proc p 
		JOIN pg_namespace n ON p.pronamespace = n.oid 
		WHERE n.nspname = 'public' AND p.proname = '%s'
	)`, functionName)
	rows, err := client.QueryContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var exists bool
	if rows.Next() {
		err = rows.Scan(&exists)
		if err != nil {
			return false, err
		}
	}
	return exists, nil
}

func getAllTables(ctx context.Context, client *sqlclient.Client) ([]string, error) {
	query := `SELECT table_schema || '.' || table_name
		FROM information_schema.tables
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		AND table_schema NOT LIKE 'pg_%'
		AND table_type = 'BASE TABLE'
		ORDER BY table_schema, table_name`
	rows, err := client.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(tableName, ".schema_migrations") || strings.HasSuffix(tableName, ".spatial_ref_sys") {
			continue
		}
		tables = append(tables, tableName)
	}
	return tables, nil
}

func checkTableTriggerExists(ctx context.Context, client *sqlclient.Client, tableName string, triggerName string) (bool, error) {
	query := `SELECT EXISTS(
		SELECT 1 FROM pg_trigger 
		WHERE tgname = $1 AND tgrelid = (
			SELECT c.oid FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE c.relname = split_part($2, '.', 2) AND n.nspname = split_part($2, '.', 1)
			)
		)
		`
	rows, err := client.QueryContext(ctx, query, triggerName, tableName)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var exists bool
	if rows.Next() {
		err = rows.Scan(&exists)
		if err != nil {
			return false, err
		}
	}
	return exists, nil
}

func getTableTriggerSQL(tableName string, triggerName string) (string, string) {
	up := fmt.Sprintf(`CREATE TRIGGER %s
BEFORE UPDATE ON %s
FOR EACH ROW
EXECUTE FUNCTION update_last_update_column()`, triggerName, tableName)
	down := fmt.Sprintf("DROP TRIGGER %s ON %s", triggerName, tableName)
	return up, down
}

func checkTableIsUnlogged(ctx context.Context, client *sqlclient.Client, schemaName string, tableName string) (bool, error) {
	query := `SELECT relpersistence = 'u' FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2`
	rows, err := client.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var isUnlogged bool
	if rows.Next() {
		err = rows.Scan(&isUnlogged)
		if err != nil {
			return false, err
		}
	}
	return isUnlogged, nil
}
