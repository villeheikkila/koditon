package config

import (
	"flag"
	"fmt"
	"io"
	"time"
)

type Config struct {
	Host               string
	Port               string
	ShutdownTimeout    time.Duration
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	HintatiedotBaseURL string
}

func Load(
	args []string,
	getenv func(string) string,
	stderr io.Writer,
) (Config, error) {
	dotEnv, err := loadDotEnv(defaultDotEnvPaths()...)
	if err != nil {
		return Config{}, err
	}
	env := getenvWithDotEnv(dotEnv, getenv)
	flagSet := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	host := flagSet.String("host", fallback(env("APP_HOST"), "0.0.0.0"), "HTTP listen host")
	port := flagSet.String("port", fallback(env("APP_PORT"), "8080"), "HTTP listen port")
	shutdown := flagSet.String("shutdown-timeout", fallback(env("APP_SHUTDOWN_TIMEOUT"), "10s"), "graceful shutdown timeout (e.g. 5s)")
	dbHost := flagSet.String("db-host", fallback(env("DB_HOST"), "localhost"), "Database host")
	dbPort := flagSet.String("db-port", fallback(env("DB_PORT"), "5432"), "Database port")
	dbUser := flagSet.String("db-user", fallback(env("DB_USER"), "postgres"), "Database user")
	dbPassword := flagSet.String("db-password", fallback(env("DB_PASSWORD"), "postgres"), "Database password")
	dbName := flagSet.String("db-name", fallback(env("DB_NAME"), "koditon"), "Database name")
	dbSSLMode := flagSet.String("db-sslmode", fallback(env("DB_SSLMODE"), "disable"), "Database SSL mode")
	hintatiedotBaseURL := flagSet.String("hintatiedot-base-url", env("HINTATIEDOT_BASE_URL"), "Hintatiedot base API URL")
	if err := flagSet.Parse(args[1:]); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}
	timeout, err := time.ParseDuration(*shutdown)
	if err != nil {
		return Config{}, fmt.Errorf("invalid shutdown timeout %q: %w", *shutdown, err)
	}
	return Config{
		Host:               *host,
		Port:               *port,
		ShutdownTimeout:    timeout,
		DBHost:             *dbHost,
		DBPort:             *dbPort,
		DBUser:             *dbUser,
		DBPassword:         *dbPassword,
		DBName:             *dbName,
		DBSSLMode:          *dbSSLMode,
		HintatiedotBaseURL: *hintatiedotBaseURL,
	}, nil
}

func (c Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func fallback(value, def string) string {
	if value == "" {
		return def
	}
	return value
}
