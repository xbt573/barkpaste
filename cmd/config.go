package cmd

import "time"

type Config struct {
	Listen   string   `mapstructure:"listen"`
	Database Database `mapstructure:"database"`
	Settings Settings `mapstructure:"settings"`
}

type Settings struct {
	TTL       time.Duration `mapstructure:"ttl"`
	Limit     uint          `mapstructure:"limit"`
	BodyLimit uint          `mapstructure:"bodylimit"`
	Token     string        `mapstructure:"token"`
}

type Database struct {
	Type DatabaseType `mapstructure:"type"`
	URI  string       `mapstructure:"uri"`
}

type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgresql"
	SQLite     DatabaseType = "sqlite"
)
