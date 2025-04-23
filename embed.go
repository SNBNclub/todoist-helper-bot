package embeded

import (
	"embed"
)

//go:embed sql/*.sql
var SQLEmbedFS embed.FS
