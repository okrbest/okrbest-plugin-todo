package sqlstore

import "embed"

//go:embed migrations/*.sql
var Assets embed.FS
