package model

type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var AllPermissions []*Permission

var PermissionGetUsers *Permission

func initializePermissions() {
	PermissionGetUsers = &Permission{
		ID:          "get_users",
		Name:        "Get Users",
		Description: "gets user list",
	}

	AllPermissions = append(AllPermissions, PermissionGetUsers)
}

func init() {
	initializePermissions()
}
