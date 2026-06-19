package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Port int    `mapstructure:"port" env:"SERVER_PORT"`
	Host string `mapstructure:"host" env:"SERVER_HOST"`
	Mode string `mapstructure:"mode" env:"SERVER_MODE"`
}

type BitbucketConfig struct {
	BaseURL        string `mapstructure:"base_url"        env:"BITBUCKET_BASE_URL"`
	DatacenterURL  string `mapstructure:"datacenter_url"  env:"BITBUCKET_DATACENTER_URL"`
	Token          string `mapstructure:"token"           env:"BITBUCKET_TOKEN"`
	Username       string `mapstructure:"username"        env:"BITBUCKET_USERNAME"`
	AppPassword    string `mapstructure:"app_password"    env:"BITBUCKET_APP_PASSWORD"`
}

type GitHubConfig struct {
	Token   string `mapstructure:"token"    env:"GITHUB_TOKEN"`
	BaseURL string `mapstructure:"base_url" env:"GITHUB_BASE_URL"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"  env:"LOG_LEVEL"`
	Format string `mapstructure:"format" env:"LOG_FORMAT"`
	Output string `mapstructure:"output" env:"LOG_OUTPUT"`
}

type AnalysisConfig struct {
	Architecture       bool `mapstructure:"architecture"`
	DependencyAnalysis bool `mapstructure:"dependency_analysis"`
	MigrationAnalysis  bool `mapstructure:"migration_analysis"`
	MaxDepth           int  `mapstructure:"max_depth"`
	TimeoutSeconds     int  `mapstructure:"timeout_seconds"`
}

type ObservabilityConfig struct {
	Enabled         bool   `mapstructure:"enabled"          env:"OTEL_ENABLED"`
	ServiceName     string `mapstructure:"service_name"     env:"OTEL_SERVICE_NAME"`
	TraceEndpoint   string `mapstructure:"trace_endpoint"   env:"OTEL_TRACE_ENDPOINT"`
	MetricEndpoint  string `mapstructure:"metric_endpoint"  env:"OTEL_METRIC_ENDPOINT"`
}

type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Bitbucket     BitbucketConfig     `mapstructure:"bitbucket"`
	GitHub        GitHubConfig        `mapstructure:"github"`
	Logging       LoggingConfig       `mapstructure:"logging"`
	Analysis      AnalysisConfig      `mapstructure:"analysis"`
	Observability ObservabilityConfig `mapstructure:"observability"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.mode", "stdio")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("analysis.max_depth", 10)
	v.SetDefault("analysis.timeout_seconds", 30)
	v.SetDefault("analysis.architecture", true)
	v.SetDefault("analysis.dependency_analysis", true)
	v.SetDefault("analysis.migration_analysis", true)
	v.SetDefault("observability.enabled", true)
	v.SetDefault("observability.service_name", "pr-analyzer-mcp")

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}
