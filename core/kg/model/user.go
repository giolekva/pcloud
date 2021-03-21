package model

import (
	"regexp"
	"unicode"

	"github.com/pkg/errors"
)

const (
	userNameMaxLength = 64
	userNameMinLength = 1
)

// User contains the details about the user.
type User struct {
	ID                 string `json:"id"`
	CreateAt           int64  `json:"create_at,omitempty"`
	UpdateAt           int64  `json:"update_at,omitempty"`
	DeleteAt           int64  `json:"delete_at"`
	Username           string `json:"username"`
	Password           string `json:"password,omitempty"`
	LastPasswordUpdate int64  `json:"last_password_update,omitempty"`
}

// IsValid validates the user and returns an error if it isn't configured
// correctly.
func (u *User) IsValid() error {
	if !isValidID(u.ID) {
		return invalidUserError("id", "")
	}

	if u.CreateAt == 0 {
		return invalidUserError("create_at", u.ID)
	}

	if u.UpdateAt == 0 {
		return invalidUserError("update_at", u.ID)
	}

	if !isValidUsername(u.Username) {
		return invalidUserError("username", u.ID)
	}

	return nil
}

func (u *User) Clone() *User {
	user := *u
	return &user
}

func isValidID(value string) bool {
	if len(value) != 26 {
		return false
	}

	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}

	return true
}

func invalidUserError(fieldName string, userID string) error {
	return errors.Errorf("Invalid User field: %s; id: %s", fieldName, userID)
}

func isValidUsername(s string) bool {
	if len(s) < userNameMinLength || len(s) > userNameMaxLength {
		return false
	}

	validUsernameChars := regexp.MustCompile(`^[a-z0-9\.\-_]+$`)

	return validUsernameChars.MatchString(s)
}
