package postgres

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type Connection struct {
	db *sql.DB
}

func Open(databaseURL string) (*Connection, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	return &Connection{db: db}, nil
}

func (c *Connection) Name() string {
	return "postgresql"
}

func (c *Connection) Check(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *Connection) Migrate(ctx context.Context, directory string) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, c.db, directory)
}

func (c *Connection) Close() error {
	return c.db.Close()
}

func (c *Connection) DB() *sql.DB {
	return c.db
}
