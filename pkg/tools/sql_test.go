package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadQuery(t *testing.T) {
	sqlFiles, err := findSQLFiles()
	if err != nil {
		t.Fatalf("Failed to find SQL files: %v", err)
	}

	assert.NotEmpty(t, sqlFiles, "No SQL files found for testing")

	for _, fileName := range sqlFiles {
		t.Run("Load "+fileName, func(t *testing.T) {
			result, err := LoadQuery(fileName)
			assert.NoError(t, err)
			assert.NotEmpty(t, result, "SQL content should not be empty")
		})
	}

	t.Run("Non-existent file", func(t *testing.T) {
		_, err := LoadQuery("nonexistent.sql")
		assert.Error(t, err)
	})
}

func findSQLFiles() ([]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	sqlDir := filepath.Join(filepath.Dir(filepath.Dir(wd)), "sql")

	entries, err := os.ReadDir(sqlDir)
	if err != nil {
		return nil, err
	}

	var fileNames []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			fileNames = append(fileNames, entry.Name())
		}
	}

	return fileNames, nil
}
