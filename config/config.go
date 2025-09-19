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

	// PostgreSQL 数据库配置
	PostgreSQL struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		SSLMode  string `yaml:"ssl_mode"`
		MaxConns int    `yaml:"max_conns"`
		MinConns int    `yaml:"min_conns"`
	} `yaml:"postgresql"`

	// River 队列配置
	River struct {
		Enable               bool                   `yaml:"enable"`
		Workers              int                    `yaml:"workers"`
		Queues               map[string]QueueConfig `yaml:"queues"`
		PollOnly             bool                   `yaml:"poll_only"`               // 仅轮询，不处理任务
		InsertOnly           bool                   `yaml:"insert_only"`             // 仅插入任务
		RescueStuckJobsAfter string                 `yaml:"rescue_stuck_jobs_after"` // 如 "1h"
		MaxAttempts          int                    `yaml:"max_attempts"`            // 最大重试次数
	} `yaml:"river"`
}

// QueueConfig 队列配置
type QueueConfig struct {
	MaxWorkers int `yaml:"max_workers"` // 最大工作协程数
	Priority   int `yaml:"priority"`    // 队列优先级
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
