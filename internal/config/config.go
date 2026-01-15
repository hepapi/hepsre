package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AlertManager    AlertManagerConfig    `mapstructure:"alertmanager"`
	Kubernetes      KubernetesConfig      `mapstructure:"kubernetes"`
	LogCollection   LogCollectionConfig   `mapstructure:"log_collection"`
	EventCollection EventCollectionConfig `mapstructure:"event_collection"`
	LLM             LLMConfig             `mapstructure:"llm"`
	Agent           AgentConfig           `mapstructure:"agent"`
	Server          ServerConfig          `mapstructure:"server"`
	Database        DatabaseConfig        `mapstructure:"database"`
}

type AlertManagerConfig struct {
	URL          string        `mapstructure:"url"`
	PollInterval time.Duration `mapstructure:"poll_interval"`
}

type KubernetesConfig struct {
	Kubeconfig string `mapstructure:"kubeconfig"`
	Context    string `mapstructure:"context"`
}

type LogCollectionConfig struct {
	DefaultLookback time.Duration `mapstructure:"default_lookback"`
	MaxLookback     time.Duration `mapstructure:"max_lookback"`
	TailLines       int64         `mapstructure:"tail_lines"`
	IncludePrevious bool          `mapstructure:"include_previous"`
}

type EventCollectionConfig struct {
	DefaultLookback time.Duration `mapstructure:"default_lookback"`
	MaxLookback     time.Duration `mapstructure:"max_lookback"`
	EventTypes      []string      `mapstructure:"event_types"`
}

type LLMConfig struct {
	Provider    string  `mapstructure:"provider"`
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float32 `mapstructure:"temperature"`
}

type AgentConfig struct {
	MaxParallelFetches int           `mapstructure:"max_parallel_fetches"`
	AnalysisTimeout    time.Duration `mapstructure:"analysis_timeout"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("alertmanager.poll_interval", "30s")
	v.SetDefault("log_collection.default_lookback", "1h")
	v.SetDefault("llm.provider", "anthropic")
	v.SetDefault("llm.model", "claude-sonnet-4-5")
	v.SetDefault("llm.max_tokens", 4096)
	v.SetDefault("llm.temperature", 0.2)
	v.SetDefault("database.path", "./hepsre.db")

	// Read from environment variables
	v.AutomaticEnv()

	// Read config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Override with environment variable if set
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" && config.LLM.Provider == "openai" {
		config.LLM.APIKey = apiKey
	}

	return &config, nil
}
