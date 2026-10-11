package gormdb

import (
	"context"
	"reflect"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestWAMessageDeleteByIdQuery(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=localhost user=test dbname=test",
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	var query string
	var args []any
	if err := db.Callback().Delete().After("gorm:delete").Register("test:capture_delete", func(db *gorm.DB) {
		query = db.Statement.SQL.String()
		args = append([]any(nil), db.Statement.Vars...)
	}); err != nil {
		t.Fatal(err)
	}
	if err := NewWAMessageRepository(db, nil).DeleteById(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	if want := `DELETE FROM "wa"."messages" WHERE id = $1`; query != want {
		t.Fatalf("query=%q, want %q", query, want)
	}
	if !reflect.DeepEqual(args, []any{int32(42)}) {
		t.Fatalf("unexpected delete arguments: %#v", args)
	}
}
