package sqlstore

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/store"
)

type SqlUserStore struct {
	*SqlStore

	// usersQuery is a starting point for all queries that return one or more Users.
	usersQuery sq.SelectBuilder
}

var _ store.UserStore = SqlUserStore{}

func newSqlUserStore(sqlStore *SqlStore) store.UserStore {
	us := &SqlUserStore{
		SqlStore: sqlStore,
	}

	// note: we are providing field names explicitly here to maintain order of columns (needed when using raw queries)
	us.usersQuery = us.getQueryBuilder().
		Select("u.ID", "u.CreateAt", "u.UpdateAt", "u.DeleteAt", "u.Username", "u.Password", "u.Email", "u.EmailVerified", "u.FirstName", "u.LastName", "u.LastPasswordUpdate").
		From("Users u")

	schema := `CREATE TABLE IF NOT EXISTS Users (
			id VARCHAR(26) PRIMARY KEY,
			create_at INTEGER,
			update_at INTEGER,
			delete_at INTEGER,
			username VARCHAR(64) UNIQUE,
			password VARCHAR(128) NULL,
			email VARCHAR(128) UNIQUE,
			email_verified BOOL NOT NULL DEFAULT FALSE,
			first_name VARCHAR(64),
			last_name VARCHAR(64),
			last_password_update INTEGER);`

	us.db.MustExec(schema)
	return us
}

func (us SqlUserStore) Save(user *model.User) (*model.User, error) {
	return nil, nil
}

func (us SqlUserStore) Get(id string) (*model.User, error) {
	return nil, nil
}

func (us SqlUserStore) GetAll() ([]*model.User, error) {
	return nil, nil
}
