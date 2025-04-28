package bot

import (
	"embed"
)

//go:embed sql/*.sql
var SQLEmbedFS embed.FS
