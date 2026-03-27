package telemetry

import (
	"strings"

	"github.com/exaring/otelpgx"
)

func NewPGXTracer(opts ...otelpgx.Option) *otelpgx.Tracer {
	defaultOpts := []otelpgx.Option{
		otelpgx.WithTrimSQLInSpanName(),
		otelpgx.WithSpanNameFunc(defaultSpanName),
	}
	defaultOpts = append(defaultOpts, opts...)
	return otelpgx.NewTracer(defaultOpts...)
}

func defaultSpanName(stmt string) string {
	if name, ok := sqlcQueryName(stmt); ok {
		return name
	}
	trimmed := firstSQLTokenSource(stmt)
	if trimmed == "" {
		return "query"
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "query"
	}
	return strings.ToLower(parts[0])
}

func sqlcQueryName(stmt string) (string, bool) {
	for line := range strings.SplitSeq(stmt, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "--") {
			return "", false
		}
		comment := strings.TrimSpace(strings.TrimPrefix(trimmed, "--"))
		if strings.HasPrefix(comment, "name:") {
			parts := strings.Fields(comment)
			if len(parts) >= 2 {
				return parts[1], true
			}
		}
	}
	return "", false
}

func firstSQLTokenSource(stmt string) string {
	for line := range strings.SplitSeq(stmt, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		return trimmed
	}
	return ""
}
