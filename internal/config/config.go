package config

import (
	"flag"
	"fmt"
	"io"
	"time"
)

type Config struct {
	Host            string
	Port            string
	ShutdownTimeout time.Duration
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
	if err := flagSet.Parse(args[1:]); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}
	timeout, err := time.ParseDuration(*shutdown)
	if err != nil {
		return Config{}, fmt.Errorf("invalid shutdown timeout %q: %w", *shutdown, err)
	}
	return Config{
		Host:            *host,
		Port:            *port,
		ShutdownTimeout: timeout,
	}, nil
}

func fallback(value, def string) string {
	if value == "" {
		return def
	}
	return value
}
