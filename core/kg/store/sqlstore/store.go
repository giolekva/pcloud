package sqlstore

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/store"
	"github.com/jmoiron/sqlx"
)

type SqlStore struct {
	db     *sqlx.DB
	stores SqlStoreStores
	config *model.SQLConfig
}

var _ store.Store = &SqlStore{}

type SqlStoreStores struct {
	user    store.UserStore
	session store.SessionStore
}

func New(config model.SQLConfig) *SqlStore {
	store := &SqlStore{
		config: &config,
	}

	store.initConnection()

	store.stores.user = newSqlUserStore(store)
	store.stores.session = newSqlSessionStore(store)

	return store
}

func (ss *SqlStore) initConnection() {
	ss.db = sqlx.MustConnect(ss.config.DriverName, ss.config.DataSource)
}

func (ss *SqlStore) getQueryBuilder() sq.StatementBuilderType {
	return sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(ss.db)
}

func (ss *SqlStore) User() store.UserStore {
	return ss.stores.user
}

func (ss *SqlStore) Session() store.SessionStore {
	return ss.stores.session
}
