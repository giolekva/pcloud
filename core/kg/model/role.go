package model

import "strings"

const (
	RoleNameMaxLength        = 64
	RoleDescriptionMaxLength = 1024
)

type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	CreateAt    int64    `json:"create_at"`
	UpdateAt    int64    `json:"update_at"`
	DeleteAt    int64    `json:"delete_at"`
	Permissions []string `json:"permissions"`
}

func (r *Role) IsValid() bool {
	if !isValidID(r.ID) {
		return false
	}

	return r.IsValidWithoutID()
}

func (r *Role) IsValidWithoutID() bool {
	if r.Name == "" || len(r.Name) > RoleNameMaxLength {
		return false
	}

	if strings.TrimLeft(r.Name, "abcdefghijklmnopqrstuvwxyz0123456789_") != "" {
		return false
	}

	if len(r.Description) > RoleDescriptionMaxLength {
		return false
	}

	check := func(perms []*Permission, permission string) bool {
		for _, p := range perms {
			if permission == p.ID {
				return true
			}
		}
		return false
	}
	for _, permission := range r.Permissions {
		permissionValidated := check(AllPermissions, permission)
		if !permissionValidated {
			return false
		}
	}

	return true
}
