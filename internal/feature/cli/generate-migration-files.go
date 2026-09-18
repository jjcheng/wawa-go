package feature_cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dao"
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/setup"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // register the postgres driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // register the file:// source
	"gorm.io/gorm"
)

// this does not migrate but generate the migration files (.up.sql and .down.sql)
// it will first create a temp database using the existing migration files, then compare with local db to generate additional migration files if needed
func GenerateMigrationFiles(ctx context.Context) []string {
	log.Println("========GENERATING MIGRATION FILES========")
	migrationFolder := "migration"
	abs, err := filepath.Abs(migrationFolder)
	if err != nil {
		panic(err.Error())
	}
	if !helper.CheckExists(abs) {
		migrationFolder = "../../" + migrationFolder
		rAbs, err := filepath.Abs(migrationFolder)
		if err != nil {
			panic(err.Error())
		}
		abs = rAbs
		if !helper.CheckExists(abs) {
			panic("migration folder not found")
		}
	}
	logger := service.NewLogger()
	sourceUnitOfWork, err := setup.SetupDatabase(cfg.Default().Database.DSN(), logger)
	if err != nil {
		panic(err.Error())
	}
	//1. migrate the existing .up and .down to migration database
	log.Println("creating migration database")
	setup.CreateMigrationDB(cfg.Default().Database)
	mig, err := migrate.New("file://"+abs, cfg.Default().Database.MigrateURL())
	if err != nil {
		panic("migrate.New failed: " + err.Error())
	}
	defer func() {
		log.Println("dropping migration database")
		setup.DropMigrationDB(cfg.Default().Database)
	}()
	log.Println("migrating up")
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		//if there is no .sql file, skip
		containsSql, _ := helper.CheckContainsFileWithExtension(abs, ".sql")
		if containsSql {
			panic("migrate.Up failed: " + err.Error())
		}
	}
	//2. migrate table schemas
	schemaUps, schemaDowns := migrateSchema(ctx)
	//3. migrate table data
	migrateUnitOfWork, err := setup.SetupDatabase(cfg.Default().Database.MigrateDSN(), logger)
	if err != nil {
		panic(err.Error())
	}
	dataUps, dataDowns, tables, maxIds := migrateData(ctx, migrateUnitOfWork, sourceUnitOfWork)
	//check if any has changed
	if len(schemaUps) == 0 && len(dataUps) == 0 {
		log.Println("no schema or data changed")
		return nil
	}
	schemaFiles := createSchemaFiles(migrationFolder, schemaUps, schemaDowns)
	dataFiles := createDataFiles(migrationFolder, dataUps, dataDowns, tables, maxIds)
	return append(schemaFiles, dataFiles...)
}

// data migration
func migrateData(ctx context.Context, targetUnitOfWork repository.UnitOfWork, sourceUnitOfWork repository.UnitOfWork) ([]string, []string, []string, []int32) {
	ups := []string{}
	downs := []string{}
	tables := []string{}
	maxIds := []int32{}
	//4. migrate table data
	dataRunners := []func() error{
		// account
		func() error {
			return migrateTableData[dao_account.User](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
		func() error {
			return migrateTableData[dao_account.Setting](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
		func() error {
			return migrateTableData[dao_account.Session](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
		// wa
		func() error {
			return migrateTableData[dao_wa.BusinessPortfolio](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
		func() error {
			return migrateTableData[dao_wa.BusinessAccount](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
		func() error {
			return migrateTableData[dao_wa.PhoneNumber](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
		func() error {
			return migrateTableData[dao_wa.MessageStatusEvent](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
		func() error {
			return migrateTableData[dao_wa.Message](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
		func() error {
			return migrateTableData[dao_wa.SampleTemplate](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
		// customer
		func() error {
			return migrateTableData[dao_customer.Customer](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
		func() error {
			return migrateTableData[dao_customer.Campaign](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
		func() error {
			return migrateTableData[dao_customer.CampaignRecipient](ctx, targetUnitOfWork.DB(), sourceUnitOfWork.DB(), &ups, &downs, &tables, &maxIds)
		},
	}
	//run them now
	for _, run := range dataRunners {
		if err := run(); err != nil {
			panic(err.Error())
		}
	}
	return ups, downs, tables, maxIds
}

func migrateTableData[T dao.DAO](ctx context.Context, migrationDB *gorm.DB, localDB *gorm.DB, allUps *[]string, allDowns *[]string, tables *[]string, maxIds *[]int32) error {
	var requestObject T
	//get existing (from migration files) and new (from test database)
	var localDBItems []T
	if err := localDB.WithContext(ctx).Order("id").Find(&localDBItems).Error; err != nil {
		return err
	}
	var migrationDBItems []T
	if migrationDB.Migrator().HasTable(&requestObject) {
		if err := migrationDB.WithContext(ctx).Order("id").Find(&migrationDBItems).Error; err != nil {
			return err
		}
	}
	//find new or updated
	var maxExistingId int32
	if len(migrationDBItems) > 0 {
		existingIds := helper.Map(migrationDBItems, func(s T) int32 {
			return s.Base().Id
		})
		helper.Sort(existingIds, func(id1, id2 int32) int {
			if id1 > id2 {
				return 1
			}
			if id1 < id2 {
				return -1
			}
			return 0
		})
		maxExistingId = existingIds[len(existingIds)-1]
	}
	ups := []string{}
	downs := []string{}
	//insert, update
	for _, newItem := range localDBItems {
		if newItem.Base().Id > maxExistingId { //insert
			up := generateInsertSQL(localDB, newItem)
			ups = append(ups, up)
			down := generateDeleteSQL(localDB, newItem)
			downs = append(downs, down)
		} else { //if exists, update
			if len(migrationDBItems) > 0 {
				if existing := helper.First(migrationDBItems, func(e T) bool {
					return e.Base().Id == newItem.Base().Id
				}); existing != nil && helper.GetTimeStamp((*existing).Base().LastUpdate) != helper.GetTimeStamp((newItem.Base().LastUpdate)) {
					// log.Println(helper.GetTimeStamp((*existing).Base().LastUpdate))
					// log.Println(helper.GetTimeStamp((newItem.Base().LastUpdate)))
					up := generateUpdateSQL(localDB, newItem)
					ups = append(ups, up)
					down := generateUpdateSQL(migrationDB, *existing)
					downs = append(downs, down)
				}
			}
		}
	}
	//delete old ones
	for _, existingItem := range migrationDBItems {
		if !helper.Any(localDBItems, func(s T) bool {
			return s.Base().Id == existingItem.Base().Id
		}) {
			up := generateDeleteSQL(migrationDB, existingItem)
			ups = append(ups, up)
			down := generateInsertSQL(migrationDB, existingItem)
			downs = append(downs, down)
		}
	}
	//add to list
	if len(ups) > 0 {
		disableTriggers := fmt.Sprintf("ALTER TABLE %s DISABLE TRIGGER USER", requestObject.TableName())
		enableTriggers := fmt.Sprintf("ALTER TABLE %s ENABLE TRIGGER USER", requestObject.TableName())
		// ups
		*allUps = append(*allUps, disableTriggers)
		*allUps = append(*allUps, ups...)
		*allUps = append(*allUps, enableTriggers)
		// downs
		*allDowns = append(*allDowns, disableTriggers)
		*allDowns = append(*allDowns, downs...)
		*allDowns = append(*allDowns, enableTriggers)
		// tables
		*tables = append(*tables, requestObject.TableName())
		// max id for roll back
		*maxIds = append(*maxIds, maxExistingId)
	}
	return nil
}

func generateInsertSQL(db *gorm.DB, T dao.DAO) string {
	statement := db.Session(&gorm.Session{DryRun: true}).Create(T).Statement
	sql := db.Dialector.Explain(statement.SQL.String(), statement.Vars...)
	return sql
}

func generateDeleteSQL(db *gorm.DB, T dao.DAO) string {
	statement := db.Session(&gorm.Session{DryRun: true}).Delete(T).Statement
	sql := db.Dialector.Explain(statement.SQL.String(), statement.Vars...)
	return sql
}

func generateUpdateSQL(db *gorm.DB, T dao.DAO) string {
	statement := db.Session(&gorm.Session{DryRun: true}).Updates(T).Statement
	sql := db.Dialector.Explain(statement.SQL.String(), statement.Vars...)
	return sql
}

// utilities
func getVersionPrefix(migrationFolder string) string {
	//find the latest version number
	files, err := os.ReadDir(migrationFolder)
	if err != nil {
		panic(err.Error())
	}
	maxVersion := 0
	maxVersionString := ""
	for _, file := range files {
		fileName := file.Name()
		//only check for xxx.sql files
		if !strings.HasSuffix(fileName, ".sql") {
			continue
		}
		index := strings.Split(fileName, "_")[0]
		indexInt, err := strconv.Atoi(index)
		if err != nil {
			panic(err.Error())
		}
		if indexInt > maxVersion {
			maxVersion = indexInt
			maxVersionString = index
		}
	}
	//add 1 to max version
	maxVersion += 1
	if maxVersionString == "" { //no up down yet
		maxVersionString = "000001"
	} else {
		maxVersionString = helper.PadLeft(fmt.Sprint(maxVersion), len(maxVersionString), '0')
	}
	return maxVersionString
}

func createSchemaFiles(migrationFolder string, ups []string, downs []string) []string {
	if len(ups) == 0 && len(downs) == 0 {
		return nil
	}
	//save
	upSql := `BEGIN;
COMMIT;`
	if len(ups) > 0 {
		upSql = fmt.Sprintf(
			`BEGIN;
%v;
COMMIT;
ANALYZE;`, strings.Join(ups, ";\n"))
	}
	downSql := `BEGIN;
COMMIT;`
	if len(downs) > 0 {
		downSql = fmt.Sprintf(
			`BEGIN;
%v;
COMMIT;`, strings.Join(downs, ";\n"))
	}
	versionPrefix := getVersionPrefix(migrationFolder)
	//generate up.sql
	upFileName := fmt.Sprintf("%s/%s_schema.up.sql", migrationFolder, versionPrefix)
	err := helper.WriteToFile(upSql, upFileName)
	if err != nil {
		panic(err.Error())
	}
	log.Printf("- generated %s\n", upFileName)
	//generate down.sql
	downFileName := fmt.Sprintf("%s/%s_schema.down.sql", migrationFolder, versionPrefix)
	err = helper.WriteToFile(downSql, downFileName)
	if err != nil {
		panic(err.Error())
	}
	log.Printf("- generated %s\n", downFileName)
	return []string{upFileName, downFileName}
}

func generateSequenceResetCommands(tables []string) []string {
	var commands []string
	commands = append(commands, "-- Reset identity sequences after inserting records with explicit IDs")
	commands = append(commands, "-- This ensures auto-increment starts from the correct value after existing data")
	for _, table := range tables {
		// Use PostgreSQL's setval with pg_get_serial_sequence for more reliable sequence reset
		command := fmt.Sprintf(`SELECT setval(pg_get_serial_sequence('%s', 'id'), (SELECT MAX(id) FROM %s))`, table, table)
		commands = append(commands, command)
	}
	return commands
}

func generateSequenceResetCommandsForRollback(tables []string, maxIds []int32) []string {
	var commands []string
	commands = append(commands, "-- Reset identity sequences to the previous max id after rollback")
	commands = append(commands, "-- This ensures clean state after removing data")
	for i := range tables {
		// Use PostgreSQL's setval with pg_get_serial_sequence to reset to 1
		command := fmt.Sprintf(`SELECT setval(pg_get_serial_sequence('%s', 'id'), %d, false)`, tables[i], maxIds[i])
		commands = append(commands, command)
	}
	return commands
}

func createDataFiles(migrationFolder string, ups []string, downs []string, tables []string, maxIds []int32) []string {
	if len(ups) == 0 && len(downs) == 0 {
		return nil
	}
	versionPrefix := getVersionPrefix(migrationFolder)
	upSql := `BEGIN;
COMMIT;`
	if len(ups) > 0 {
		// Add sequence reset commands after data inserts
		sequenceResets := generateSequenceResetCommands(tables)
		allUps := append(ups, sequenceResets...)
		upSql = fmt.Sprintf(
			`BEGIN;
%v;

COMMIT;
ANALYZE;`, strings.Join(allUps, ";\n"))
	}
	downSql := `BEGIN;
COMMIT;`
	if len(downs) > 0 {
		// Add sequence reset commands after data deletions in rollback
		sequenceResets := generateSequenceResetCommandsForRollback(tables, maxIds)
		allDowns := append(downs, sequenceResets...)
		downSql = fmt.Sprintf(
			`BEGIN;
%v;
COMMIT;`, strings.Join(allDowns, ";\n"))
	}
	//generate up.sql
	upFileName := fmt.Sprintf("%s/%s_data.up.sql", migrationFolder, versionPrefix)
	err := helper.WriteToFile(upSql, upFileName)
	if err != nil {
		panic(err.Error())
	}
	log.Printf("- generated %s\n", upFileName)
	//generate down.sql
	downFileName := fmt.Sprintf("%s/%s_data.down.sql", migrationFolder, versionPrefix)
	err = helper.WriteToFile(downSql, downFileName)
	if err != nil {
		panic(err.Error())
	}
	log.Printf("- generated %s\n", downFileName)
	return []string{upFileName, downFileName}
}
