package storage

import "github.com/google/uuid"

type StorageAdapter interface {
	GetLastServerKnowledge(budgetId uuid.UUID) (int64, error)
	SetLastServerKnowledge(budgetId uuid.UUID, serverKnowledge int64) error
}
