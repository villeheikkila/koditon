package schema

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestQueriesDoNotEraseNullableTextProjections(t *testing.T) {
	t.Parallel()
	queriesDir := filepath.Join("..", "..", "..", "db", "queries")
	pattern := regexp.MustCompile(`(?i)COALESCE\(\s*[a-z_][a-z0-9_\.]*\s*,\s*''(?:::text)?\s*\)(?:::text)?\s+AS\s+[a-z_][a-z0-9_]*`)
	var violations []string
	err := filepath.WalkDir(queriesDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".sql" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range pattern.FindAllStringIndex(string(data), -1) {
			line := 1 + strings.Count(string(data[:match[0]]), "\n")
			violations = append(violations, filepath.ToSlash(path)+":"+strconv.Itoa(line))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan sql queries: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("nullable text projections must not use COALESCE(..., ''): %s", strings.Join(violations, ", "))
	}
}
