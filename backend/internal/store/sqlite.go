package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/juanmafx/drug-categories/backend/internal/domain"
)

type SQLite struct {
	db *sql.DB
}

func OpenSQLite(ctx context.Context, path string) (*SQLite, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	query := url.Values{}
	query.Set("mode", "rw")
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	dsn := url.URL{Scheme: "file", Path: absolutePath, RawQuery: query.Encode()}

	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	var tableCount int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_schema WHERE type = 'table' AND name = 'categories'`).Scan(&tableCount)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("inspect database: %w", err)
	}
	if tableCount != 1 {
		_ = db.Close()
		return nil, errors.New("database is missing the categories table; apply db/schema.sql first")
	}
	return &SQLite{db: db}, nil
}

func (s *SQLite) Close() error {
	return s.db.Close()
}

func (s *SQLite) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description FROM categories ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	categories := []domain.Category{}
	for rows.Next() {
		var category domain.Category
		if err := rows.Scan(&category.ID, &category.Name, &category.Description); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return categories, nil
}

func (s *SQLite) Create(ctx context.Context, input domain.NewCategory) (domain.Category, error) {
	created := domain.Category{Name: input.Name, Description: input.Description}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO categories (name, description) VALUES (?, ?) RETURNING id`,
		input.Name, input.Description,
	).Scan(&created.ID)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
			return domain.Category{}, domain.ErrDuplicateCategory
		}
		return domain.Category{}, fmt.Errorf("insert category: %w", err)
	}
	return created, nil
}
