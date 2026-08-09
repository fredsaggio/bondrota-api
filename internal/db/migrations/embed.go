package migrations

import "embed"

// FS contains the SQL migrations used by the application and integration tests.
//
//go:embed *.sql
var FS embed.FS
