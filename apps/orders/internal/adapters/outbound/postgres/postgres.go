package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/chouaib-skitou/streamweave/apps/orders/internal/adapters/outbound/postgres/generated"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type Connection struct{ db *sql.DB }

func Open(databaseURL string) (*Connection, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
	return &Connection{db: db}, nil
}

func (c *Connection) Name() string                    { return "postgresql" }
func (c *Connection) Check(ctx context.Context) error { return c.db.PingContext(ctx) }
func (c *Connection) Close() error                    { return c.db.Close() }
func (c *Connection) DB() *sql.DB                     { return c.db }

func (c *Connection) Migrate(ctx context.Context, directory string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	const migrationLock int64 = 838927451
	queries := generated.New(c.db)
	if err := queries.AcquireMigrationLock(ctx, migrationLock); err != nil {
		return err
	}
	defer func() { _ = queries.ReleaseMigrationLock(context.Background(), migrationLock) }()
	return goose.UpContext(ctx, c.db, directory)
}
