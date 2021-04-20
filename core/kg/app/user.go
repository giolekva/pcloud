package app

import (
	"net/http"

	"github.com/giolekva/pcloud/core/kg/log"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

const HeaderToken = "token"

// GetUser returns user
func (a *App) GetUser(userID string) (*model.User, error) {
	user, err := a.store.User().Get(userID)
	if err != nil {
		return nil, errors.Wrap(err, "can't get user from store")
	}
	return user, nil
}

// CreateUser creates a user. For now it is used only for creation of the very first user
func (a *App) CreateUser(user *model.User) (*model.User, error) {
	if !a.isFirstUserAccount() {
		return nil, errors.New("not a first user")
	}

	updatedUser, err := a.store.User().Save(user)
	if err != nil {
		return nil, errors.Wrap(err, "can't save user to the DB")
	}
	return updatedUser, nil
}

//GetUsers returns list of users
func (a *App) GetUsers(page, perPage int) ([]*model.User, error) {
	users, err := a.store.User().GetAllWithOptions(page, perPage)
	if err != nil {
		return nil, errors.Wrap(err, "can't get users with options from store")
	}
	return users, nil
}

func (a *App) isFirstUserAccount() bool {
	count, err := a.store.User().Count()
	if err != nil {
		a.logger.Error("error fetching first user account", log.Err(err))
	}
	return count > 0
}

func (a *App) AuthenticateUserForLogin(userID, username, password string) (*model.User, error) {
	var user *model.User
	var err error
	switch {
	case userID != "":
		user, err = a.store.User().Get(userID)
		if err != nil {
			return nil, errors.Wrap(err, "can't get user from store")
		}
	case username != "":
		user, err = a.store.User().GetByUsername(username)
		if err != nil {
			return nil, errors.Wrapf(err, "can't get user by username")
		}
	default:
		return nil, errors.New("can't authenticate user for login")
	}

	if err := a.checkLogin(user, password); err != nil {
		return nil, errors.Wrapf(err, "login error")
	}
	return user, nil
}

func (a *App) checkLogin(user *model.User, password string) error {
	if user.IsDisabled() {
		return errors.New("user is disabled")
	}
	if !comparePassword(user.Password, password) {
		return errors.New("incorrect password")
	}
	return nil
}

func (a *App) DoLogin(w http.ResponseWriter, r *http.Request, user *model.User) error {
	session := &model.Session{
		UserID: user.ID,
	}
	session.SetExpireInDays(a.config.App.SessionLengthInDays)

	session, err := a.CreateSession(session)
	if err != nil {
		return errors.Wrap(err, "can't create a session")
	}

	w.Header().Set(HeaderToken, session.Token)
	a.SetSession(session)

	return nil
}

func (a *App) SetSession(s *model.Session) {
	a.session = *s
}

func (a *App) CreateSession(session *model.Session) (*model.Session, error) {
	session.Token = ""
	session, err := a.store.Session().Save(session)
	if err != nil {
		return nil, errors.Wrap(err, "can't save the session")
	}
	return session, nil
}

// HashPassword hashes user's password
func HashPassword(password string) string {
	if password == "" {
		return ""
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		panic(err)
	}

	return string(hash)
}

// comparePassword compares the hash
func comparePassword(hash string, password string) bool {
	if password == "" || hash == "" {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
