package soft

type Repository struct {
	Name       string `json:"name"`
	Repository string `json:"repo"`
	Private    bool   `json:"private"`
	Note       string `json:"note"`
}

type User struct {
	Name       string   `json:"name"`
	Admin      bool     `json:"admin"`
	PublicKeys []string `json:"public-keys"`
}

type Config struct {
	Name         string       `json:"name"`
	Host         string       `json:"host"`
	Port         int          `json:"port"`
	AnonAccess   string       `json:"anon-access"`
	AllowKeyless bool         `json:"allow-keyless"`
	Repositories []Repository `json:"repos"`
	Users        []User       `json:"users"`
}

func DefaultConfig(adminKeys []string) Config {
	return Config{
		Name:         "PCloud",
		Host:         "localhost",
		Port:         22,
		AnonAccess:   "no-access",
		AllowKeyless: false,
		Repositories: []Repository{
			{
				Name:       "Home",
				Repository: "config",
				Private:    true,
				Note:       "Configuration for PCloud SoftServe deployment",
			},
		},
		Users: []User{
			{
				Name:       "Admin",
				Admin:      true,
				PublicKeys: adminKeys,
			},
		},
	}
}
