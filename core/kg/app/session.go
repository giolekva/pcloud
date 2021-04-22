package app

import (
	"github.com/giolekva/pcloud/core/kg/common"
	"github.com/giolekva/pcloud/core/kg/model"
	"github.com/pkg/errors"
)

func (a *App) CreateSession(userID string) (*model.Session, error) {
	session := &model.Session{
		UserID: userID,
	}
	session.SetExpireInDays(a.config.App.SessionLengthInDays)
	session.Token = ""
	session, err := a.store.Session().Save(session)
	if err != nil {
		return nil, errors.Wrap(err, "can't save the session")
	}
	return session, nil
}

func (a *App) Session() *model.Session {
	return &a.session
}

func (a *App) RevokeSession(sessionID string) error {
	if err := a.store.Session().Remove(sessionID); err != nil {
		return errors.Wrap(err, "can't remove session")
	}
	return nil
}

func (a *App) GetSession(token string) (*model.Session, error) {
	session, err := a.store.Session().Get(token)
	if err != nil {
		return nil, errors.Wrap(err, "can't get session")
	}
	if session == nil || session.ID == "" || session.IsExpired() {
		return nil, errors.New("session is nil or expired")
	}
	if a.config.App.SessionIdleTimeoutInMinutes > 0 {
		timeout := int64(a.config.App.SessionIdleTimeoutInMinutes) * 1000 * 60
		if (common.GetMillis() - session.LastActivityAt) > timeout {
			return nil, errors.New("session idle timeout")
		}
	}
	return session, nil
}

func (a *App) SetSession(s *model.Session) {
	a.session = *s
}
