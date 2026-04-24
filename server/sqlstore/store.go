package sqlstore

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost/server/public/model"
)

// SQLStore holds the sqlx database connection and query builder.
type SQLStore struct {
	db      *sqlx.DB
	builder sq.StatementBuilderType
}

// New constructs a new instance of SQLStore using the plugin API to access the master database.
func New(pluginAPI PluginAPIClient) (*SQLStore, error) {
	origDB, err := pluginAPI.Store.GetMasterDB()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get master DB")
	}

	if pluginAPI.Store.DriverName() != model.DatabaseDriverPostgres {
		return nil, errors.New("only PostgreSQL is supported")
	}

	db := sqlx.NewDb(origDB, pluginAPI.Store.DriverName())
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	return &SQLStore{
		db:      db,
		builder: builder,
	}, nil
}

type queryer interface {
	sqlx.Queryer
}

type builder interface {
	ToSql() (string, []interface{}, error)
}

type execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	DriverName() string
}

type queryExecer interface {
	queryer
	execer
}

func (s *SQLStore) getBuilder(q sqlx.Queryer, dest interface{}, b builder) error {
	sqlString, args, err := b.ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build sql")
	}

	sqlString = s.db.Rebind(sqlString)

	return sqlx.Get(q, dest, sqlString, args...)
}

func (s *SQLStore) selectBuilder(q sqlx.Queryer, dest interface{}, b builder) error {
	sqlString, args, err := b.ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to build sql")
	}

	sqlString = s.db.Rebind(sqlString)

	return sqlx.Select(q, dest, sqlString, args...)
}

func (s *SQLStore) exec(e execer, sqlString string, args ...interface{}) (sql.Result, error) {
	sqlString = s.db.Rebind(sqlString)
	return e.Exec(sqlString, args...)
}

func (s *SQLStore) execBuilder(e execer, b builder) (sql.Result, error) {
	sqlString, args, err := b.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build sql")
	}

	return s.exec(e, sqlString, args...)
}

func (s *SQLStore) finalizeTransaction(tx *sqlx.Tx) {
	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		logrus.WithError(err).Error("Failed to rollback transaction")
	}
}
