package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Config represents the configuration for XXL-JOB executor
type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`

	XXLJob struct {
		AdminAddresses   []string `yaml:"admin_addresses"`
		AppName          string   `yaml:"app_name"`
		AccessToken      string   `yaml:"access_token"`
		LogPath          string   `yaml:"log_path"`
		LogRetentionDays int      `yaml:"log_retention_days"`
	} `yaml:"xxljob"`

	Executor struct {
		IP       string `yaml:"ip"`
		Port     int    `yaml:"port"`
		ServerID string `yaml:"server_id"`
	} `yaml:"executor"`

	Database map[string]struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		Charset  string `yaml:"charset"`
	} `yaml:"database"`

	Queue struct {
		Enable             bool `yaml:"enable"`
		ConfigSyncInterval int  `yaml:"config_sync_interval"`
	} `yaml:"queue"`
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
