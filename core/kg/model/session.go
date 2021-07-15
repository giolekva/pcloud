package model

import "time"

// Session contains the user session details.
type Session struct {
	ID             string `json:"id"`
	Token          string `json:"token"`
	CreateAt       int64  `json:"create_at"`
	ExpiresAt      int64  `json:"expires_at"`
	LastActivityAt int64  `json:"last_activity_at"`
	UserID         string `json:"user_id"`
}

func (s *Session) Sanitize() {
	s.Token = ""
}

func (s *Session) IsExpired() bool {
	if s.ExpiresAt <= 0 {
		return false
	}

	if time.Now().Unix() > s.ExpiresAt {
		return true
	}

	return false
}

// SetExpireInDays sets the session's expiry the specified number of days
func (s *Session) SetExpireInDays(days int) {
	if s.CreateAt == 0 {
		s.ExpiresAt = time.Now().Unix() + (1000 * 60 * 60 * 24 * int64(days))
	} else {
		s.ExpiresAt = s.CreateAt + (1000 * 60 * 60 * 24 * int64(days))
	}
}
