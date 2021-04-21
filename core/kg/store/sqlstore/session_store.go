package sqlstore

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/store"
)

type SqlSessionStore struct {
	*SqlStore

	// sessionsQuery is a starting point for all queries that return one or more Sessions.
	sessionsQuery sq.SelectBuilder
}

var _ store.SessionStore = SqlSessionStore{}

func newSqlSessionStore(sqlStore *SqlStore) store.SessionStore {
	ss := &SqlSessionStore{
		SqlStore: sqlStore,
	}

	// note: we are providing field names explicitly here to maintain order of columns (needed when using raw queries)
	ss.sessionsQuery = ss.getQueryBuilder().
		Select("s.ID", "s.Token", "s.CreateAt", "s.ExpiresAt", "s.LastActivityAt", "s.UserID").
		From("Sessions s")

	schema := `CREATE TABLE IF NOT EXISTS Sessions (
			id VARCHAR(26) PRIMARY KEY,
			token VARCHAR(26), 
			create_at INTEGER,
			expires_at INTEGER,
			last_activity_at INTEGER,
			user_id VARCHAR(26);`

	ss.db.MustExec(schema)
	return ss
}

func (ss SqlSessionStore) Save(session *model.Session) (*model.Session, error) {
	return nil, nil
}

func (ss SqlSessionStore) Remove(sessionID string) error {
	return nil
}
