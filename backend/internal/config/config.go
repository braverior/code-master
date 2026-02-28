package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Feishu   FeishuConfig   `mapstructure:"feishu"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Codegen  CodegenConfig  `mapstructure:"codegen"`
	Encrypt  EncryptConfig  `mapstructure:"encrypt"`
	AIChat   AIChatConfig   `mapstructure:"ai_chat"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	Charset  string `mapstructure:"charset"`
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=UTC",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.Charset)
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type FeishuConfig struct {
	AppID       string    `mapstructure:"app_id"`
	AppSecret   string    `mapstructure:"app_secret"`
	RedirectURI string    `mapstructure:"redirect_uri"`
	Bot         BotConfig `mapstructure:"bot"`
}

type BotConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	EncryptKey        string `mapstructure:"encrypt_key"`
	VerificationToken string `mapstructure:"verification_token"`
}

type AIChatConfig struct {
	BaseURL    string `mapstructure:"base_url"`
	APIKey     string `mapstructure:"api_key"`
	Model      string `mapstructure:"model"`
	MaxHistory int    `mapstructure:"max_history"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type CodegenConfig struct {
	MaxWorkers       int                `mapstructure:"max_workers"`
	MaxTurns         int                `mapstructure:"max_turns"`
	TimeoutMinutes   int                `mapstructure:"timeout_minutes"`
	WorkDir          string             `mapstructure:"work_dir"`
	UseLocalGit      bool               `mapstructure:"use_local_git"`
	GitDomainMapping []GitDomainMapping `mapstructure:"git_domain_mapping"`
}

type GitDomainMapping struct {
	From string `mapstructure:"from"`
	To   string `mapstructure:"to"`
}

type EncryptConfig struct {
	AESKey string `mapstructure:"aes_key"`
}

var Global *Config

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	Global = &cfg
	return &cfg, nil
}
