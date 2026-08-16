package data_sqlite

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	. "github.com/infrago/base"
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

func TestSQLiteDialectNormalizesTimestampBindingsToUTC(t *testing.T) {
	dialect := sqliteDialect{}
	field := Var{Type: "datetime"}
	instant := time.Date(2026, time.August, 16, 19, 30, 0, 123456789, time.UTC)
	local := instant.In(time.FixedZone("Pacific", -7*60*60))

	boundUTC, ok := dialect.BindValue(field, instant)
	if !ok {
		t.Fatal("UTC timestamp was not bound")
	}
	boundLocal, ok := dialect.BindValue(field, local)
	if !ok {
		t.Fatal("offset timestamp was not bound")
	}
	if boundUTC != boundLocal || boundUTC != "2026-08-16T19:30:00.123456789Z" {
		t.Fatalf("timestamps were not normalized: UTC=%#v local=%#v", boundUTC, boundLocal)
	}
	wholeSecond, _ := dialect.BindValue(field, instant.Truncate(time.Second))
	fractionalSecond, _ := dialect.BindValue(field, instant.Truncate(time.Second).Add(time.Nanosecond))
	if wholeSecond.(string) >= fractionalSecond.(string) {
		t.Fatalf("fixed-width timestamps do not sort chronologically: %q >= %q", wholeSecond, fractionalSecond)
	}
}

func TestSQLiteNormalizedTimestampsCompareChronologically(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE outbox (available_at DATETIME, expires_at DATETIME)"); err != nil {
		t.Fatalf("create outbox failed: %v", err)
	}
	dialect := sqliteDialect{}
	field := Var{Type: "datetime"}
	now := time.Date(2026, time.August, 16, 19, 30, 0, 0, time.UTC)
	available, _ := dialect.BindValue(field, now.Add(-time.Minute).In(time.FixedZone("Pacific", -7*60*60)))
	expires, _ := dialect.BindValue(field, now.Add(time.Minute).In(time.FixedZone("Pacific", -7*60*60)))
	queryTime, _ := dialect.BindValue(field, now)
	if _, err := db.Exec("INSERT INTO outbox(available_at, expires_at) VALUES (?, ?)", available, expires); err != nil {
		t.Fatalf("insert outbox failed: %v", err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM outbox WHERE available_at <= ? AND expires_at > ?", queryTime, queryTime).Scan(&count); err != nil {
		t.Fatalf("query outbox failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("normalized timestamp window matched %d rows, want 1", count)
	}
}
