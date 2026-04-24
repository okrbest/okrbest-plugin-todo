package sqlstore

import (
	"github.com/blang/semver"
	"github.com/pkg/errors"
)

const systemDatabaseVersionKey = "DatabaseVersion"

func LatestVersion() semver.Version {
	return migrations[len(migrations)-1].toVersion
}

func (s *SQLStore) GetCurrentVersion() (semver.Version, error) {
	currentVersionStr, err := s.getSystemValue(s.db, systemDatabaseVersionKey)
	if err != nil {
		return semver.Version{}, errors.Wrapf(err, "failed retrieving the DatabaseVersion key from the TODO_System table")
	}

	if currentVersionStr == "" {
		return semver.Version{}, nil
	}

	currentSchemaVersion, err := semver.Parse(currentVersionStr)
	if err != nil {
		return semver.Version{}, errors.Wrapf(err, "unable to parse current schema version")
	}

	return currentSchemaVersion, nil
}

func (s *SQLStore) SetCurrentVersion(e queryExecer, currentVersion semver.Version) error {
	return s.setSystemValue(e, systemDatabaseVersionKey, currentVersion.String())
}
