package data_sqlite

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/infrago/data"
)

func TestSQLiteDialectClassifiesErrors(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT UNIQUE)"); err != nil {
		t.Fatalf("create table failed: %v", err)
	}
	if _, err := db.Exec("INSERT INTO users(name) VALUES (?)", "alice"); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	_, err = db.Exec("INSERT INTO users(name) VALUES (?)", "alice")
	if err == nil {
		t.Fatalf("expected duplicate error")
	}
	got := data.Error("insert", data.ErrInvalidUpdate, (sqliteDialect{}).ClassifyError(err))
	if !errors.Is(got, data.ErrDuplicate) {
		t.Fatalf("expected duplicate classification, got %v", got)
	}
}
