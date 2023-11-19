package storage

type StorageAdapter interface {
	GetLastServerKnowledge(budgetId string) (int64, error)
	SetLastServerKnowledge(budgetId string, serverKnowledge int64) error
}
