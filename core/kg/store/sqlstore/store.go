package sqlstore

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/store"
	"github.com/jmoiron/sqlx"
)

type SqlStore struct {
	db       *sqlx.DB
	stores   SqlStoreStores
	settings *model.SqlSettings
}

var _ store.Store = &SqlStore{}

type SqlStoreStores struct {
	user store.UserStore
}

func New(settings model.SqlSettings) *SqlStore {
	store := &SqlStore{
		settings: &settings,
	}

	store.initConnection()

	store.stores.user = newSqlUserStore(store)

	return store
}

func (ss *SqlStore) initConnection() {
	ss.db = sqlx.MustConnect(ss.settings.DriverName, ss.settings.DataSource)
}

func (ss *SqlStore) getQueryBuilder() sq.StatementBuilderType {
	return sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
}

func (ss *SqlStore) User() store.UserStore {
	return ss.stores.user
}
