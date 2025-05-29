package tools

import (
	"fmt"
	"path/filepath"

	embeded "example.com/bot/storage"
)

func LoadQuery(filename string) (string, error) {
	// filename := filepath.Base(filePath)

	embeddedPath := filepath.Join("queries", filename)

	data, err := embeded.SQLEmbedFS.ReadFile(embeddedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read SQL file %s: %w", filename, err)
	}

	return string(data), nil
}
