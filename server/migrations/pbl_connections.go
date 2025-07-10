package migrations

import (
	"github.com/pocketbase/dbx"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		_, err := db.NewQuery(`
        CREATE TABLE {{_pbl_connections}} (
            [[id]] TEXT PRIMARY KEY NOT NULL,
            [[name]] TEXT NOT NULL,
            [[type]] TEXT NOT NULL,
            [[config]] JSON NOT NULL,
            [[created]] TEXT DEFAULT (strftime('%Y-%m-%d %H:%M:%fZ')) NOT NULL,
            [[updated]] TEXT DEFAULT (strftime('%Y-%m-%d %H:%M:%fZ')) NOT NULL
        );
        CREATE INDEX _pbl_connections_type_idx ON {{_pbl_connections}} ([[type]]);
        `).Execute()
		return err
	}, func(db dbx.Builder) error {
		_, err := db.NewQuery("DROP TABLE {{_pbl_connections}};").Execute()
		return err
	})
}
