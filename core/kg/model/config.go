package model

const (
	databaseDriverPostgres = "postgres"
	defaultDataSource      = "postgres://user:test@localhost/pcloud_test?sslmode=disable&connect_timeout=10"
)

type Config struct {
	SqlSettings SqlSettings
}

func NewConfig() *Config {
	config := &Config{}
	config.SetDefaults()
	return config
}

func (c *Config) SetDefaults() {
	c.SqlSettings.SetDefaults()
}

type SqlSettings struct {
	DriverName string `access:"environment,write_restrictable,cloud_restrictable"`
	DataSource string `access:"environment,write_restrictable,cloud_restrictable"`
}

func (s *SqlSettings) SetDefaults() {
	if s.DriverName == "" {
		s.DriverName = databaseDriverPostgres
	}

	if s.DataSource == "" {
		s.DataSource = defaultDataSource
	}
}
