package store

import (
	"context"
	"slices"
	"sync"

	"github.com/juanmafx/drug-categories/backend/internal/domain"
)

type Memory struct {
	mu         sync.RWMutex
	categories []domain.Category
	names      map[string]struct{}
}

func NewMemory(seed ...domain.Category) *Memory {
	memory := &Memory{
		categories: slices.Clone(seed),
		names:      make(map[string]struct{}, len(seed)),
	}
	for _, category := range seed {
		memory.names[domain.NameKey(category.Name)] = struct{}{}
	}
	return memory
}

func DefaultSeed() []domain.Category {
	return []domain.Category{{ID: 1, Name: "Analgesics", Description: "Pain relief drugs"}}
}

func (m *Memory) List(ctx context.Context) ([]domain.Category, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Clone(m.categories), nil
}

func (m *Memory) Create(ctx context.Context, input domain.NewCategory) (domain.Category, error) {
	if err := ctx.Err(); err != nil {
		return domain.Category{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := domain.NameKey(input.Name)
	if _, exists := m.names[key]; exists {
		return domain.Category{}, domain.ErrDuplicateCategory
	}
	nextID := 1
	if count := len(m.categories); count > 0 {
		nextID = m.categories[count-1].ID + 1
	}
	created := domain.Category{ID: nextID, Name: input.Name, Description: input.Description}
	m.categories = append(m.categories, created)
	m.names[key] = struct{}{}
	return created, nil
}
