package storage

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type localStorageAdapter struct{}

const storageFile = "storage.yml"

type budgetData struct {
	BudgetId            string `yaml:"budgetId"`
	LastServerKnowledge int64  `yaml:"lastServerKnowledge"`
}

// Creates a StorageAdapter which stores data in a yaml file. Intended mostly for prototyping or running in environments
// without "proper" KV storage mechanisms.
func NewLocalStorageAdapter() StorageAdapter {
	return &localStorageAdapter{}
}

func (l *localStorageAdapter) GetLastServerKnowledge(budgetId string) (int64, error) {
	data, err := l.readData(budgetId)
	if err != nil {
		return 0, err
	}

	for _, d := range data {
		if d.BudgetId == budgetId {
			return d.LastServerKnowledge, nil
		}
	}

	return 0, fmt.Errorf("No budget found with id %v", budgetId)
}

func (l *localStorageAdapter) SetLastServerKnowledge(budgetId string, serverKnowledge int64) error {
	var data []budgetData

	if _, err := os.Stat(storageFile); err == nil {
		data, err = l.readData(budgetId)
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
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	defer encoder.Close()

	return encoder.Encode(data)
}

func (l *localStorageAdapter) readData(budgetId string) ([]budgetData, error) {
	f, err := os.Open(storageFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var data []budgetData
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
