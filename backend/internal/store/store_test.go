package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/juanmafx/drug-categories/backend/internal/domain"
)

func newSQLiteStore(t *testing.T) *SQLite {
	t.Helper()
	path := filepath.Join(t.TempDir(), "categories.db")
	schema, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	opened, err := OpenSQLite(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	return opened
}

func repositories(t *testing.T) map[string]Repository {
	t.Helper()
	return map[string]Repository{
		"memory": NewMemory(DefaultSeed()...),
		"sqlite": newSQLiteStore(t),
	}
}

func TestRepositoryContract(t *testing.T) {
	ctx := context.Background()
	for label, repository := range repositories(t) {
		t.Run(label, func(t *testing.T) {
			initial, err := repository.List(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(initial) != 1 || initial[0].Name != "Analgesics" {
				t.Fatalf("unexpected seed: %+v", initial)
			}

			created, err := repository.Create(ctx, domain.NewCategory{Name: "Antivirals", Description: "Treat viral infections"})
			if err != nil {
				t.Fatal(err)
			}
			if created.ID != 2 {
				t.Fatalf("got id %d, want 2", created.ID)
			}

			_, err = repository.Create(ctx, domain.NewCategory{Name: "ANTIVIRALS"})
			if !errors.Is(err, domain.ErrDuplicateCategory) {
				t.Fatalf("got %v, want ErrDuplicateCategory", err)
			}

			listed, err := repository.List(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(listed) != 2 || listed[1] != created {
				t.Fatalf("unexpected list: %+v", listed)
			}
		})
	}
}

func TestRepositoryTreatsInjectionPayloadAsData(t *testing.T) {
	ctx := context.Background()
	payload := "x'); DROP TABLE categories; --"
	for label, repository := range repositories(t) {
		t.Run(label, func(t *testing.T) {
			created, err := repository.Create(ctx, domain.NewCategory{Name: payload})
			if err != nil {
				t.Fatal(err)
			}
			if created.Name != payload {
				t.Fatalf("got %q", created.Name)
			}
			listed, err := repository.List(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(listed) != 2 {
				t.Fatalf("got %d rows", len(listed))
			}
		})
	}
}

func TestRepositoryConcurrentCreatesAssignUniqueIDs(t *testing.T) {
	ctx := context.Background()
	for label, repository := range repositories(t) {
		t.Run(label, func(t *testing.T) {
			const workers = 25
			var group sync.WaitGroup
			failures := make(chan error, workers)
			for index := range workers {
				group.Add(1)
				go func() {
					defer group.Done()
					_, err := repository.Create(ctx, domain.NewCategory{Name: fmt.Sprintf("Category %d", index)})
					failures <- err
				}()
			}
			group.Wait()
			close(failures)
			for err := range failures {
				if err != nil {
					t.Fatal(err)
				}
			}
			listed, err := repository.List(ctx)
			if err != nil {
				t.Fatal(err)
			}
			seen := map[int]struct{}{}
			for _, category := range listed {
				seen[category.ID] = struct{}{}
			}
			if len(listed) != workers+1 || len(seen) != workers+1 {
				t.Fatalf("got %d rows and %d unique ids", len(listed), len(seen))
			}
		})
	}
}

func TestMemoryListReturnsCopy(t *testing.T) {
	memory := NewMemory(DefaultSeed()...)
	listed, _ := memory.List(context.Background())
	listed[0].Name = "Mutated"
	again, _ := memory.List(context.Background())
	if again[0].Name != "Analgesics" {
		t.Fatalf("store was mutated: %+v", again)
	}
}

func TestMemoryHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	memory := NewMemory()
	if _, err := memory.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if _, err := memory.Create(ctx, domain.NewCategory{Name: "A"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestOpenSQLiteRequiresSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE unrelated (id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	_, err = OpenSQLite(context.Background(), path)
	if err == nil || !strings.Contains(err.Error(), "db/schema.sql") {
		t.Fatalf("got %v", err)
	}
}

func TestOpenSQLiteDoesNotCreateMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	if _, err := OpenSQLite(context.Background(), path); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("database file was created: %v", err)
	}
}

func TestSchemaEnforcesConstraints(t *testing.T) {
	opened := newSQLiteStore(t)
	statements := []string{
		`INSERT INTO categories (name) VALUES ('')`,
		`INSERT INTO categories (name) VALUES (' padded ')`,
		`INSERT INTO categories (name) VALUES ('` + strings.Repeat("a", 121) + `')`,
		`INSERT INTO categories (name, description) VALUES ('Long', '` + strings.Repeat("a", 1001) + `')`,
	}
	for _, statement := range statements {
		if _, err := opened.db.Exec(statement); err == nil {
			t.Fatalf("expected constraint failure for %.50s", statement)
		}
	}
}
