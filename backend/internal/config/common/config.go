package common

import (
	"strings"
	"time"
)

// Common config for all services

type Config struct {
	ConfigFilePath   string `mapstructure:"configFilePath"`
	MessageSizeLimit int    `mapstructure:"messageSizeLimit"`
	MaxMemoryUsage   uint64 `mapstructure:"maxMemoryUsage"`
	MemoryLimitMB    uint64 `mapstructure:"memoryLimitMB"`
	// Interservice authentication keys (loaded from /vault/secrets/static.json when VAULT_ENABLED=true)
	GoInterserviceAPIKey     string `mapstructure:"goInterserviceApiKey"`
	PythonInterserviceAPIKey string `mapstructure:"pythonInterserviceApiKey"`
}

type Configer interface {
	GetConfigPath() string
}

func (c *Config) GetConfigPath() string {
	return c.ConfigFilePath
}

func (c *Config) GoInterserviceApiKey() string {
	return c.GoInterserviceAPIKey
}

func (c *Config) PythonInterserviceApiKey() string {
	return c.PythonInterserviceAPIKey
}

type Postgres struct {
	Postgres        string `mapstructure:"string"`
	ApplicationName string `mapstructure:"applicationName"`
}

func (cfg *Postgres) String() string {
	str := cfg.Postgres
	if str == "" {
		return ""
	}
	if !strings.Contains(cfg.Postgres, "application_name") {
		if strings.Contains(cfg.Postgres, "?") {
			str += "&"
		} else {
			str += "?"
		}
		name := cfg.ApplicationName
		if name == "" {
			name = "worker"
		}
		str += "application_name=" + name
	}
	return str
}

type Redshift struct {
	ConnectionString string `mapstructure:"connectionString"`
	Host             string `mapstructure:"host"`
	Port             int    `mapstructure:"port"`
	User             string `mapstructure:"user"`
	Password         string `mapstructure:"password"`
	Database         string `mapstructure:"database"`
	Bucket           string `mapstructure:"bucket"`
}

type Clickhouse struct {
	Enabled              bool          `mapstructure:"enabled"`
	URL                  string        `mapstructure:"url"`
	UrlHTTP              string        `mapstructure:"urlHttp"`
	Database             string        `mapstructure:"database"`
	UserName             string        `mapstructure:"username"`
	Password             string        `mapstructure:"password"`
	LegacyUserName       string        `mapstructure:"legacyUsername"`
	LegacyPassword       string        `mapstructure:"legacyPassword"`
	MaxOpenConns         int           `mapstructure:"maxOpenConns"`
	MaxIdleConns         int           `mapstructure:"maxIdleConns"`
	ConnMaxLifetime      time.Duration `mapstructure:"connMaxLifetime"`
	CompressionAlgo      string        `mapstructure:"compressionAlgo"`
	DEBUG                bool          `mapstructure:"debug"`
	MaxExecutionTime     int           `mapstructure:"maxExecutionTime"`
	UseTLS               bool          `mapstructure:"useTls"`
	TLSSkipVerify        bool          `mapstructure:"tlsSkipVerify"`
	TLSCertificatePath   string        `mapstructure:"tlsCertPath"`
	TLSKeyPath           string        `mapstructure:"tlsKeyPath"`
	TLSCACertificatePath string        `mapstructure:"tlsCaPath"`
}

func (cfg *Clickhouse) GetTrimmedURL() string {
	chUrl := strings.TrimPrefix(cfg.URL, "tcp://")
	chUrl = strings.TrimSuffix(chUrl, "/default")
	return chUrl
}

func (cfg *Clickhouse) GetTrimmedUrlHTTP() string {
	chUrl := strings.TrimPrefix(cfg.UrlHTTP, "tcp://")
	chUrl = strings.TrimSuffix(chUrl, "/default")
	return chUrl
}

type ElasticSearch struct {
	URLs   string `mapstructure:"urls"`
	UseAWS bool   `mapstructure:"useAws"`
}

func (cfg *ElasticSearch) GetURLs() []string {
	return strings.Split(cfg.URLs, ",")
}

type HTTP struct {
	HTTPHost                string        `mapstructure:"host"`
	HTTPPort                string        `mapstructure:"port"`
	HTTPTimeout             time.Duration `mapstructure:"timeout"`
	JsonSizeLimit           int64         `mapstructure:"jsonSizeLimit"`
	JsonWithDataSizeLimit   int64         `mapstructure:"jsonWithDataSizeLimit"`
	UseAccessControlHeaders bool          `mapstructure:"useCors"`
	JWTSecret               string        `mapstructure:"jwtSecret"`
	JWTSpotSecret           string        `mapstructure:"jwtSpotSecret"`
	JWTExpiration           time.Duration `mapstructure:"jwtExpiration"`
	JWTIssuer               string        `mapstructure:"jwtIssuer"`
}

type RateLimiter struct {
	Rate            int           `mapstructure:"rate"`
	Burst           int           `mapstructure:"burst"`
	CleanupInterval time.Duration `mapstructure:"cleanupInterval"`
	MaxIdleTime     time.Duration `mapstructure:"maxIdleTime"`
}
