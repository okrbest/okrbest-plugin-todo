package sqlstore

import (
	"github.com/blang/semver"
	"github.com/pkg/errors"
)

// RunMigrations applies all pending schema migrations in sequence.
// The caller should hold a cluster mutex if there is a danger of this being run
// on multiple servers at once.
func (s *SQLStore) RunMigrations() error {
	currentSchemaVersion, err := s.GetCurrentVersion()
	if err != nil {
		return errors.Wrapf(err, "failed to get the current schema version")
	}

	if currentSchemaVersion.LT(LatestVersion()) {
		if err := s.runMigrationsLegacy(currentSchemaVersion); err != nil {
			return errors.Wrapf(err, "failed to complete migrations")
		}
	}

	return nil
}

func (s *SQLStore) runMigrationsLegacy(originalSchemaVersion semver.Version) error {
	currentSchemaVersion := originalSchemaVersion
	for _, migration := range migrations {
		if !currentSchemaVersion.EQ(migration.fromVersion) {
			continue
		}

		if err := s.applyMigration(migration); err != nil {
			return err
		}

		currentSchemaVersion = migration.toVersion
	}

	return nil
}

func (s *SQLStore) applyMigration(migration Migration) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "could not begin transaction")
	}
	defer s.finalizeTransaction(tx)

	if err := migration.migrationFunc(tx, s); err != nil {
		return errors.Wrapf(err, "error executing migration from version %s to version %s",
			migration.fromVersion.String(), migration.toVersion.String())
	}

	if err := s.SetCurrentVersion(tx, migration.toVersion); err != nil {
		return errors.Wrapf(err, "failed to set the current version to %s", migration.toVersion.String())
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "could not commit transaction")
	}

	return nil
}
