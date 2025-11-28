package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func loadDotEnv(paths ...string) (map[string]string, error) {
	env := make(map[string]string)

	for _, p := range paths {
		if p == "" {
			continue
		}
		file, err := os.Open(p)
		if errors.Is(err, os.ErrNotExist) {
			continue // silently ignore missing files
		}
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", p, err)
		}

		scanner := bufio.NewScanner(file)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := strings.TrimSpace(scanner.Text())

			if line == "" || strings.HasPrefix(line,

				"#") {
				continue
			}

			kv := strings.SplitN(line, "=", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("%s:%d malformed line", p, lineNo)
			}

			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])

			val = strings.Trim(val, `"'`)

			env[key] = val
		}

		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("scan %s: %w", p, err)
		}
		_ = file.Close()
	}

	return env, nil
}

func getenvWithDotEnv(dotEnv map[string]string, realGetenv func(string) string) func(string) string {
	return func(key string) string {
		if val, ok := dotEnv[key]; ok {
			return val
		}
		return realGetenv(key)
	}
}

func defaultDotEnvPaths() []string {
	cwd, err := os.Getwd()
	if err != nil {
		return []string{".env", ".env.local"}
	}
	return []string{
		filepath.Join(cwd, ".env"),
		filepath.Join(cwd, ".env.local"),
	}
}
