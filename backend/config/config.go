package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port           int    `envconfig:"PORT" default:"8080"`
	Env            string `envconfig:"ENV" default:"development"`
	DBUser         string `envconfig:"DB_USER" default:"postgres"`
	DBName         string `envconfig:"DB_NAME" default:"app"`
	DBPass         string `envconfig:"DB_PASSWORD" default:"pass"`
	DBPort         int    `envconfig:"DB_PORT" default:"5432"`
	DB_HOST        string `envconfig:"DB_HOST" default:"localhost"`
	DBUnixSocket   string `envconfig:"INSTANCE_UNIX_SOCKET" default:""`
	StaticFilePath string `envconfig:"STATIC_FILE_PATH" required:"true"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
