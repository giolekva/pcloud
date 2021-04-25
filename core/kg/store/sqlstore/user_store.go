package sqlstore

import (
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/giolekva/pcloud/core/kg/common"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/giolekva/pcloud/core/kg/store"
	"github.com/pkg/errors"
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
		Select("u.ID", "u.CreateAt", "u.UpdateAt", "u.DeleteAt", "u.Username", "u.Password", "u.LastPasswordUpdate").
		From("Users u")

	schema := `CREATE TABLE IF NOT EXISTS Users (
			id VARCHAR(26) PRIMARY KEY,
			create_at INTEGER,
			update_at INTEGER,
			delete_at INTEGER,
			username VARCHAR(64) UNIQUE,
			password VARCHAR(128) NULL,
			last_password_update INTEGER);`

	us.db.MustExec(schema)
	return us
}

func (us SqlUserStore) Save(user *model.User) (*model.User, error) {
	now := time.Now().Unix()

	updatedUser := user.Clone()

	if updatedUser.ID == "" {
		updatedUser.ID = common.NewID()
	}
	updatedUser.CreateAt = now
	updatedUser.UpdateAt = now
	updatedUser.DeleteAt = 0
	updatedUser.LastPasswordUpdate = now

	query := us.getQueryBuilder().Insert("Users").
		Columns("id", "username", "password", "create_at", "update_at", "delete_at", "last_password_update").
		Values(updatedUser.ID, updatedUser.Username, updatedUser.Password, updatedUser.CreateAt, updatedUser.UpdateAt, updatedUser.DeleteAt, updatedUser.LastPasswordUpdate)

	if _, err := query.Exec(); err != nil {
		return nil, errors.Wrap(model.ErrInternal, err.Error())
	}
	return updatedUser, nil
}

func (us SqlUserStore) Get(id string) (*model.User, error) {
	return us.getUserByCondition(sq.Eq{"id": id})
}

func (us SqlUserStore) GetAll() ([]*model.User, error) {
	query := us.usersQuery.OrderBy("Username ASC")
	rows, err := query.Query()
	if err != nil {
		return nil, errors.Wrap(model.ErrInternal, err.Error()) //TODO work on a clear db errors
	}
	return getUsersFromRows(rows)
}

func (us SqlUserStore) Count() (int64, error) {
	query := us.getQueryBuilder().
		Select("count(*)").
		From("Users").
		Where(sq.Eq{"delete_at": 0})
	row := query.QueryRow()

	var count int64
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (us SqlUserStore) GetAllWithOptions(page, perPage int) ([]*model.User, error) {
	query := us.usersQuery.Offset(uint64(page * perPage)).Limit(uint64(perPage)).OrderBy("Username ASC")
	rows, err := query.Query()
	if err != nil {
		return nil, errors.Wrap(model.ErrInternal, err.Error()) //TODO work on a clear db errors
	}
	return getUsersFromRows(rows)
}

func (us SqlUserStore) GetByUsername(username string) (*model.User, error) {
	return us.getUserByCondition(sq.Eq{"username": username})
}

func (us SqlUserStore) getUserByCondition(condition sq.Eq) (*model.User, error) {
	query := us.usersQuery.
		Where(sq.Eq{"delete_at": 0}).
		Where(condition)
	row := query.QueryRow()
	user := model.User{}

	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.CreateAt, &user.UpdateAt, &user.DeleteAt, &user.LastPasswordUpdate)
	if err != nil {
		return nil, errors.Wrap(err, "can't scan row")
	}

	return &user, nil
}

func getUsersFromRows(rows *sql.Rows) ([]*model.User, error) {
	defer rows.Close()
	results := []*model.User{}
	for rows.Next() {
		user := model.User{}
		err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.CreateAt, &user.UpdateAt, &user.DeleteAt, &user.LastPasswordUpdate)
		if err != nil {
			return nil, errors.Wrap(err, "can't scan user row")
		}
		results = append(results, &user)
	}
	return results, nil
}
