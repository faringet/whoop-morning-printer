package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strconv"
	"strings"
)

type Migration struct {
	Version  int64
	Name     string
	FileName string
}

type RunnerConfig struct {
	DB            *sql.DB
	FS            fs.FS
	Dir           string
	LockNamespace string
	LockResource  string
	Log           *slog.Logger
}

type Runner struct {
	db            *sql.DB
	fsys          fs.FS
	dir           string
	lockNamespace string
	lockResource  string
	log           *slog.Logger
}

func NewRunner(cfg RunnerConfig) (*Runner, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("migrate: db is nil")
	}
	if cfg.FS == nil {
		return nil, fmt.Errorf("migrate: fs is nil")
	}
	if strings.TrimSpace(cfg.Dir) == "" {
		return nil, fmt.Errorf("migrate: dir is required")
	}
	if strings.TrimSpace(cfg.LockNamespace) == "" {
		return nil, fmt.Errorf("migrate: lock namespace is required")
	}
	if strings.TrimSpace(cfg.LockResource) == "" {
		return nil, fmt.Errorf("migrate: lock resource is required")
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}

	return &Runner{
		db:            cfg.DB,
		fsys:          cfg.FS,
		dir:           cfg.Dir,
		lockNamespace: cfg.LockNamespace,
		lockResource:  cfg.LockResource,
		log:           cfg.Log,
	}, nil
}

func (r *Runner) Run(ctx context.Context) error {
	lockID, err := StableAdvisoryLockID(r.lockNamespace, r.lockResource)
	if err != nil {
		return fmt.Errorf("migrate: build advisory lock id: %w", err)
	}

	migrations, err := discoverMigrations(r.fsys, r.dir)
	if err != nil {
		return err
	}

	conn, err := r.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migrate: db conn: %w", err)
	}
	defer conn.Close()

	r.log.Info("acquiring migration advisory lock",
		slog.String("lock_namespace", r.lockNamespace),
		slog.String("lock_resource", r.lockResource),
		slog.Int64("lock_id", lockID),
	)

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return fmt.Errorf("migrate: advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, lockID)
	}()

	if err := ensureSchemaMigrationsTable(ctx, conn); err != nil {
		return err
	}

	applied, err := loadAppliedVersions(ctx, conn)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if _, ok := applied[migration.Version]; ok {
			r.log.Debug("migration already applied",
				slog.Int64("version", migration.Version),
				slog.String("name", migration.Name),
			)
			continue
		}

		sqlBytes, err := fs.ReadFile(r.fsys, path.Join(r.dir, migration.FileName))
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", migration.FileName, err)
		}

		query := strings.TrimSpace(string(sqlBytes))
		if query == "" {
			return fmt.Errorf("migrate: migration %s is empty", migration.FileName)
		}

		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("migrate: begin tx for %s: %w", migration.FileName, err)
		}

		if _, err := tx.ExecContext(ctx, query); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate: exec %s: %w", migration.FileName, err)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO schema_migrations(version, name, applied_at) VALUES ($1, $2, NOW())`,
			migration.Version,
			migration.Name,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate: mark applied %s: %w", migration.FileName, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrate: commit %s: %w", migration.FileName, err)
		}

		r.log.Info("migration applied",
			slog.Int64("version", migration.Version),
			slog.String("name", migration.Name),
			slog.String("file", migration.FileName),
		)
	}

	r.log.Info("migrations completed", slog.Int("count", len(migrations)))

	return nil
}

func discoverMigrations(fsys fs.FS, dir string) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("migrate: read dir %s: %w", dir, err)
	}

	migrations := make([]Migration, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}

		version, migrationName, err := parseMigrationFileName(name)
		if err != nil {
			return nil, err
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     migrationName,
			FileName: name,
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func parseMigrationFileName(name string) (int64, string, error) {
	base := strings.TrimSuffix(name, ".up.sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("migrate: invalid migration file name %q", name)
	}

	version, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("migrate: invalid migration version in %q: %w", name, err)
	}

	migrationName := strings.TrimSpace(parts[1])
	if migrationName == "" {
		return 0, "", fmt.Errorf("migrate: empty migration name in %q", name)
	}

	return version, migrationName, nil
}

func ensureSchemaMigrationsTable(ctx context.Context, conn *sql.Conn) error {
	_, err := conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    BIGINT PRIMARY KEY,
			name       TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("migrate: ensure schema_migrations table: %w", err)
	}

	return nil
}

func loadAppliedVersions(ctx context.Context, conn *sql.Conn) (map[int64]struct{}, error) {
	rows, err := conn.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("migrate: load applied versions: %w", err)
	}
	defer rows.Close()

	applied := make(map[int64]struct{})

	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("migrate: scan applied version: %w", err)
		}

		applied[version] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: applied version rows: %w", err)
	}

	return applied, nil
}
