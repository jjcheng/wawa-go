package gormdb

import (
	"context"
	"reflect"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestListMatchingByPhoneNumberIdQuery(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=localhost user=test dbname=test",
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	var query string
	var args []any
	if err := db.Callback().Query().After("gorm:query").Register("test:capture_query", func(db *gorm.DB) {
		query = db.Statement.SQL.String()
		args = append([]any(nil), db.Statement.Vars...)
	}); err != nil {
		t.Fatal(err)
	}
	repository := NewWABusinessAgentKeywordRepository(db, nil)
	messageText := "URGENT: 100% a_b 'quoted'"
	if _, err := repository.ListMatchingByPhoneNumberId(context.Background(), 42, messageText); err != nil {
		t.Fatal(err)
	}
	wantQuery := `SELECT * FROM "wa"."business_agent_keywords" WHERE phone_number_id = $1 AND char_length(keyword) <= char_length($2) AND strpos(lower($3), lower(keyword)) > 0 ORDER BY id`
	if query != wantQuery {
		t.Errorf("query = %q, want %q", query, wantQuery)
	}
	wantArgs := []any{int32(42), messageText, messageText}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Errorf("args = %#v, want %#v", args, wantArgs)
	}
}
