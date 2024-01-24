package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/Saaghh/wallet/internal/config"
	"github.com/jackc/pgx/v5"
	migrate "github.com/rubenv/sql-migrate"
	"go.uber.org/zap"
)

type Postgres struct {
	db  *pgx.Conn
	dsn string
}

//go:embed migrations
var migrations embed.FS

func New(ctx context.Context, cfg config.Config) (*Postgres, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", cfg.PGUser, cfg.PGPassword, cfg.PGHost, cfg.PGPort, cfg.PGDatabase)

	db, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx.Connect: %w", err)
	}

	err = db.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	zap.L().Info("successfully connected to db")

	return &Postgres{
		db:  db,
		dsn: dsn,
	}, nil
}

func (p *Postgres) Migrate(direction migrate.MigrationDirection) error {
	conn, err := sql.Open("pgx", p.dsn)
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}

	defer func() {
		err := conn.Close()
		if err != nil {
			zap.L().With(zap.Error(err)).Warn("conn.Close")
		}
	}()

	assetDir := func() func(string) ([]string, error) {
		return func(path string) ([]string, error) {
			dirEntry, err := migrations.ReadDir(path)
			if err != nil {
				return nil, fmt.Errorf("migrations.ReadDir: %w", err)
			}

			entries := make([]string, 0)

			for _, e := range dirEntry {
				entries = append(entries, e.Name())
			}

			return entries, nil
		}
	}()

	asset := migrate.AssetMigrationSource{
		Asset:    migrations.ReadFile,
		AssetDir: assetDir,
		Dir:      "migrations",
	}

	_, err = migrate.Exec(conn, "postgres", asset, direction)
	if err != nil {
		return fmt.Errorf("migrate.Exec: %w", err)
	}

	return nil
}