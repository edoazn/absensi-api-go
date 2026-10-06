package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"time"

	"github.com/edoazn/absensi-go/config"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func EnsureDatabase(cfg *config.Config) error {
	serverDB, err := sql.Open("mysql", cfg.MySQLServerDSN(false))
	if err != nil {
		return fmt.Errorf("open mysql server: %w", err)
	}
	defer func() { _ = serverDB.Close() }()

	if err := serverDB.Ping(); err != nil {
		return fmt.Errorf("mysql unreachable: %w", err)
	}

	stmt := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		cfg.DBName,
	)
	if _, err := serverDB.Exec(stmt); err != nil {
		return fmt.Errorf("create database %s: %w", cfg.DBName, err)
	}
	return nil
}

func Connect(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

type migrationFile struct {
	version int
	name    string
	path    string
}

var migrationNameRe = regexp.MustCompile(`^(\d+)_([a-z0-9_]+)\.up\.sql$`)

func Migrate(cfg *config.Config) error {
	mdb, err := sql.Open("mysql", cfg.MySQLServerDSN(true))
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer func() { _ = mdb.Close() }()

	if err := mdb.Ping(); err != nil {
		return fmt.Errorf("mysql unreachable: %w", err)
	}

	if _, err := mdb.Exec("USE `" + cfg.DBName + "`"); err != nil {
		return fmt.Errorf("use database %s: %w", cfg.DBName, err)
	}

	if _, err := mdb.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT UNSIGNED NOT NULL,
			applied_at DATETIME NOT NULL,
			PRIMARY KEY (version)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := loadMigrationFiles()
	if err != nil {
		return err
	}

	var applied []int
	rows, err := mdb.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("read applied migrations: %w", err)
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan applied migration: %w", err)
		}
		applied = append(applied, v)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate applied migrations: %w", err)
	}
	_ = rows.Close()

	appliedSet := make(map[int]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	for _, mf := range files {
		if appliedSet[mf.version] {
			continue
		}
		content, readErr := migrationsFS.ReadFile(mf.path)
		if readErr != nil {
			return fmt.Errorf("read migration %s: %w", mf.name, readErr)
		}
		if _, execErr := mdb.Exec(string(content)); execErr != nil {
			return fmt.Errorf("apply migration %s: %w", mf.name, execErr)
		}
		if _, recErr := mdb.Exec(
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			mf.version, time.Now(),
		); recErr != nil {
			return fmt.Errorf("record migration %s: %w", mf.name, recErr)
		}
	}
	return nil
}

func loadMigrationFiles() ([]migrationFile, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations dir: %w", err)
	}

	var files []migrationFile
	for _, e := range entries {
		m := migrationNameRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		var version int
		if _, scanErr := fmt.Sscanf(m[1], "%d", &version); scanErr != nil {
			return nil, fmt.Errorf("invalid migration version in %q", e.Name())
		}
		files = append(files, migrationFile{
			version: version,
			name:    m[2],
			path:    "migrations/" + e.Name(),
		})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })
	return files, nil
}
