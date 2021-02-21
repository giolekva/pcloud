package model

const (
	databaseDriverPostgres = "postgres"
	defaultDataSource      = "postgres://user:test@localhost/pcloud_test?sslmode=disable&connect_timeout=10"

	defaultHTTPHost         = "0.0.0.0"
	defaultHTTPPort         = 9086
	defaultHTTPReadTimeout  = 5
	defaultHTTPWriteTimeout = 10
	defaultHTTPIdleTimeout  = 120

	defaultGRPCPort = 9087
)

type Config struct {
	SQLSettings  SQLSettings
	HTTPSettings HTTPSettings
	GRPCSettings GRPCSettings
}

func NewConfig() *Config {
	config := &Config{}
	config.SetDefaults()
	return config
}

func (c *Config) SetDefaults() {
	c.SQLSettings.SetDefaults()
	c.HTTPSettings.SetDefaults()
	c.GRPCSettings.SetDefaults()
}

type SQLSettings struct {
	DriverName string
	DataSource string
}

func (s *SQLSettings) SetDefaults() {
	if s.DriverName == "" {
		s.DriverName = databaseDriverPostgres
	}

	if s.DataSource == "" {
		s.DataSource = defaultDataSource
	}
}

type HTTPSettings struct {
	Host         string
	Port         int
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
}

func (s *HTTPSettings) SetDefaults() {
	if s.Host == "" {
		s.Host = defaultHTTPHost
	}

	if s.Port == 0 {
		s.Port = defaultHTTPPort
	}

	if s.ReadTimeout == 0 {
		s.ReadTimeout = defaultHTTPReadTimeout
	}

	if s.WriteTimeout == 0 {
		s.WriteTimeout = defaultHTTPWriteTimeout
	}

	if s.IdleTimeout == 0 {
		s.IdleTimeout = defaultHTTPIdleTimeout
	}
}

type GRPCSettings struct {
	Port int
}

func (s *GRPCSettings) SetDefaults() {
	if s.Port == 0 {
		s.Port = defaultGRPCPort
	}
}
