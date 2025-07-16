package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	RabbitMQ struct {
		URL string `yaml:"url"`
	} `yaml:"rabbitmq"`
	Database struct {
		URL string `yaml:"url"`
	} `yaml:"database"`
	Workers   int    `yaml:"workers"`
	JWTSecret string `yaml:"jwt_secret"`
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	return &cfg, err
}
