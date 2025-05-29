package embeded

import (
	"embed"
)

//go:embed queries/*.sql
var SQLEmbedFS embed.FS
