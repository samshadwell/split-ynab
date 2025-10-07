package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type localStorageAdapter struct{}

const storageFile = "storage.yml"

type budgetData struct {
	BudgetId            uuid.UUID `yaml:"budgetId"`
	LastServerKnowledge int64     `yaml:"lastServerKnowledge"`
}

// Creates a StorageAdapter which stores data in a yaml file. Intended mostly for prototyping or running in environments
// without "proper" KV storage mechanisms.
func NewLocalStorageAdapter() StorageAdapter {
	return &localStorageAdapter{}
}

func (l *localStorageAdapter) GetLastServerKnowledge(ctx context.Context, budgetId uuid.UUID) (int64, error) {
	data, err := l.readData()
	if err != nil {
		return 0, err
	}

	for _, d := range data {
		if d.BudgetId == budgetId {
			return d.LastServerKnowledge, nil
		}
	}

	return 0, fmt.Errorf("no budget found with id %v", budgetId)
}

func (l *localStorageAdapter) SetLastServerKnowledge(ctx context.Context, budgetId uuid.UUID, serverKnowledge int64) (err error) {
	var data []budgetData

	if _, err := os.Stat(storageFile); err == nil {
		data, err = l.readData()
		if err != nil {
			return err
		}
	}

	found := false
	for i, d := range data {
		if d.BudgetId == budgetId {
			data[i].LastServerKnowledge = serverKnowledge
			found = true
			break
		}
	}
	if !found {
		data = append(data, budgetData{
			BudgetId:            budgetId,
			LastServerKnowledge: serverKnowledge,
		})
	}

	f, err := os.Create(storageFile)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("failed to close storage file: %w", closeErr)
		}
	}()

	encoder := yaml.NewEncoder(f)
	defer func() {
		if closeErr := encoder.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("failed to close YAML encoder: %w", closeErr)
		}
	}()

	if err = encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode storage data: %w", err)
	}

	return nil
}

func (l *localStorageAdapter) readData() (data []budgetData, err error) {
	f, err := os.Open(storageFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("failed to close storage file: %w", closeErr)
		}
	}()

	decoder := yaml.NewDecoder(f)
	if err = decoder.Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}
