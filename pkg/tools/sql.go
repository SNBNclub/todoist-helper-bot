package tools

import (
	"path/filepath"

	"example.com/bot"
)

func LoadQuery(filename string) (string, error) {
	embeddedPath := filepath.Join("sql", filename)
	data, err := bot.SQLEmbedFS.ReadFile(embeddedPath)
	return string(data), err
}
