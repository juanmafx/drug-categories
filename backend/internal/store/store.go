package store

import (
	"context"

	"github.com/juanmafx/drug-categories/backend/internal/domain"
)

type Repository interface {
	List(ctx context.Context) ([]domain.Category, error)
	Create(ctx context.Context, input domain.NewCategory) (domain.Category, error)
}
