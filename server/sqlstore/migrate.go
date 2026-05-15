package sqlstore

import (
	"context"
	"fmt"

	"github.com/mattermost/morph"
	"github.com/mattermost/morph/drivers/postgres"
	"github.com/mattermost/morph/sources/embedded"
	"github.com/pkg/errors"

	_ "github.com/lib/pq"
)

const migrationTablePrefix = "todo"

func (s *SQLStore) RunMigrations() error {
	driver, err := postgres.WithInstance(s.db.DB)
	if err != nil {
		return errors.Wrap(err, "failed to create migration driver")
	}

	assetsList, err := Assets.ReadDir("migrations")
	if err != nil {
		return errors.Wrap(err, "failed to read migration assets")
	}

	assetNames := make([]string, len(assetsList))
	for i, entry := range assetsList {
		assetNames[i] = entry.Name()
	}

	src, err := embedded.WithInstance(&embedded.AssetSource{
		Names: assetNames,
		AssetFunc: func(name string) ([]byte, error) {
			return Assets.ReadFile("migrations/" + name)
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create migration source")
	}

	engine, err := morph.New(context.Background(), driver, src,
		morph.WithLock(fmt.Sprintf("%s-migration-lock", migrationTablePrefix)),
		morph.SetMigrationTableName(fmt.Sprintf("%s_schema_migrations", migrationTablePrefix)),
		morph.SetStatementTimeoutInSeconds(100000),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create migration engine")
	}
	defer engine.Close()

	if err := engine.ApplyAll(); err != nil {
		return errors.Wrap(err, "failed to apply migrations")
	}

	return nil
}
