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
	SQL  SQLConfig
	HTTP HTTPConfig
	GRPC GRPCConfig
}

func NewConfig() *Config {
	config := &Config{}
	config.SetDefaults()
	return config
}

func (c *Config) SetDefaults() {
	c.SQL.SetDefaults()
	c.HTTP.SetDefaults()
	c.GRPC.SetDefaults()
}

type SQLConfig struct {
	DriverName string
	DataSource string
}

func (s *SQLConfig) SetDefaults() {
	if s.DriverName == "" {
		s.DriverName = databaseDriverPostgres
	}
	if s.DataSource == "" {
		s.DataSource = defaultDataSource
	}
}

type HTTPConfig struct {
	Host         string
	Port         int
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
}

func (s *HTTPConfig) SetDefaults() {
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

type GRPCConfig struct {
	Port int
}

func (s *GRPCConfig) SetDefaults() {
	if s.Port == 0 {
		s.Port = defaultGRPCPort
	}
}
