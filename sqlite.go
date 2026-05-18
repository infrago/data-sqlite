package data_sqlite

import (
	"database/sql"
	"errors"
	"strings"
	"sync/atomic"

	. "github.com/infrago/base"
	"github.com/infrago/data"
	"modernc.org/sqlite"
)

type (
	sqliteDriver struct{}

	sqliteConnection struct {
		instance *data.Instance
		db       *sql.DB
		actives  int64
	}

	sqliteDialect struct{}
)

func (d *sqliteDriver) Connect(inst *data.Instance) (data.Connection, error) {
	return &sqliteConnection{instance: inst}, nil
}

func (c *sqliteConnection) Open() error {
	dsn := strings.TrimSpace(c.instance.Config.Url)
	if dsn == "" {
		if v, ok := c.instance.Setting["dsn"].(string); ok {
			dsn = v
		}
	}
	if dsn == "" {
		dsn = "file:data.db"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return err
	}
	c.db = db
	return nil
}

func (c *sqliteConnection) Close() error {
	if c.db == nil {
		return nil
	}
	err := c.db.Close()
	c.db = nil
	return err
}

func (c *sqliteConnection) Health() data.Health {
	return data.Health{Workload: atomic.LoadInt64(&c.actives)}
}

func (c *sqliteConnection) DB() *sql.DB {
	return c.db
}

func (c *sqliteConnection) Dialect() data.Dialect {
	return sqliteDialect{}
}

func (sqliteDialect) Name() string { return "sqlite" }
func (sqliteDialect) Quote(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, `"`, ``)
	return `"` + s + `"`
}
func (sqliteDialect) Placeholder(_ int) string { return "?" }
func (sqliteDialect) SupportsILike() bool      { return false }
func (sqliteDialect) SupportsReturning() bool  { return true }
func (sqliteDialect) MaxParams() int           { return 999 }
func (sqliteDialect) ClassifyError(err error) error {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return nil
	}
	code := sqliteErr.Code()
	switch code {
	case 1555, 2067:
		return data.ErrDuplicate
	case 787:
		return data.ErrForeignKey
	case 5, 6:
		return data.ErrTimeout
	case 9:
		return data.ErrCanceled
	case 14:
		return data.ErrDriver
	default:
		if code&0xff == 19 {
			return data.ErrConflict
		}
		return nil
	}
}
func (sqliteDialect) BindValue(cfg Var, v any) (any, bool) {
	switch {
	case data.IsJSONVar(cfg):
		return data.BindJSONValue(v)
	case data.IsBinaryVar(cfg):
		return data.BindBinaryValue(v)
	case data.IsUUIDVar(cfg), data.IsDecimalVar(cfg):
		return data.BindTextValue(v)
	case data.IsTimeVar(cfg):
		return data.BindTimeValue(v)
	default:
		return nil, false
	}
}
func (sqliteDialect) DecodeValue(cfg Var, value any) (any, bool) {
	switch {
	case data.IsJSONVar(cfg):
		return data.DecodeJSONValue(value)
	case data.IsBinaryVar(cfg):
		return data.DecodeBinaryValue(value)
	case data.IsUUIDVar(cfg), data.IsDecimalVar(cfg):
		return data.DecodeTextValue(value)
	case data.IsTimeVar(cfg):
		return data.DecodeTimeValue(value)
	default:
		return nil, false
	}
}
