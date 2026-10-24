// Package migrations embeds the sequential Goose SQL schema migrations.
package migrations

import "embed"

// FS contains all SQL migration files under migrations/*.sql.
//
//go:embed *.sql
var FS embed.FS
