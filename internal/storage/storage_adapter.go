package storage

import (
	"context"

	"github.com/google/uuid"
)

type StorageAdapter interface {
	GetLastServerKnowledge(ctx context.Context, budgetId uuid.UUID) (int64, error)
	SetLastServerKnowledge(ctx context.Context, budgetId uuid.UUID, serverKnowledge int64) error
}
