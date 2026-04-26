package pretty

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"charm.land/lipgloss/v2"
)

var _ slog.Handler = (*Handler)(nil)

type Options struct {
	Level          slog.Leveler
	AddSource      bool
	TimeFormat     string
	ComponentKey   string
	ComponentWidth int
	DisableColor   bool
	Multiline      bool
	Separator      bool
	SeparatorWidth int
}

type Styles struct {
	Timestamp      lipgloss.Style
	LevelDebug     lipgloss.Style
	LevelInfo      lipgloss.Style
	LevelWarn      lipgloss.Style
	LevelError     lipgloss.Style
	Component      lipgloss.Style
	Source         lipgloss.Style
	Message        lipgloss.Style
	Key            lipgloss.Style
	Value          lipgloss.Style
	ErrorValue     lipgloss.Style
	IDValue        lipgloss.Style
	Status2xx      lipgloss.Style
	Status3xx      lipgloss.Style
	Status4xx      lipgloss.Style
	Status5xx      lipgloss.Style
	DurationFast   lipgloss.Style
	DurationMedium lipgloss.Style
	DurationSlow   lipgloss.Style
	MethodRead     lipgloss.Style
	MethodWrite    lipgloss.Style
	MethodOther    lipgloss.Style
	Separator      lipgloss.Style
}

type Handler struct {
	out    io.Writer
	opts   Options
	styles Styles
	mu     *sync.Mutex
	attrs  []slog.Attr
	groups []string
}

func DefaultOptions() Options {
	return Options{
		Level:          slog.LevelInfo,
		AddSource:      false,
		TimeFormat:     "15:04:05.000",
		ComponentKey:   "component",
		ComponentWidth: 18,
		DisableColor:   false,
		Multiline:      false,
		Separator:      false,
		SeparatorWidth: 90,
	}
}

func NewHandler(w io.Writer, opts *Options) *Handler {
	cfg := DefaultOptions()
	if opts != nil {
		if opts.Level != nil {
			cfg.Level = opts.Level
		}
		if opts.TimeFormat != "" {
			cfg.TimeFormat = opts.TimeFormat
		}
		if opts.ComponentKey != "" {
			cfg.ComponentKey = opts.ComponentKey
		}
		if opts.ComponentWidth > 0 {
			cfg.ComponentWidth = opts.ComponentWidth
		}
		if opts.SeparatorWidth > 0 {
			cfg.SeparatorWidth = opts.SeparatorWidth
		}
		cfg.AddSource = opts.AddSource
		cfg.DisableColor = opts.DisableColor
		cfg.Multiline = opts.Multiline
		cfg.Separator = opts.Separator
	}
	if w == nil {
		w = os.Stderr
	}
	return &Handler{
		out:    w,
		opts:   cfg,
		styles: defaultStyles(cfg.DisableColor),
		mu:     &sync.Mutex{},
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	if h.opts.Level == nil {
		return true
	}
	return level >= h.opts.Level.Level()
}

func (h *Handler) Handle(_ context.Context, record slog.Record) error {
	attrMap := make(map[string]slog.Value, len(h.attrs)+record.NumAttrs()+4)
	for _, attr := range h.attrs {
		h.appendAttr(attrMap, h.groups, attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		h.appendAttr(attrMap, h.groups, attr)
		return true
	})
	component := h.extractComponent(attrMap)
	orderedKeys := orderedAttrKeys(attrMap)
	now := record.Time
	if now.IsZero() {
		now = time.Now()
	}
	message := sanitizeOneLine(record.Message)
	if message == "" {
		message = "-"
	}
	line := h.renderRecord(now, record, component, message, orderedKeys, attrMap)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := lipgloss.Fprint(h.out, line.String())
	return err
}

func (h *Handler) renderRecord(ts time.Time, record slog.Record, component, message string, orderedKeys []string, attrs map[string]slog.Value) strings.Builder {
	if h.opts.Multiline {
		return h.renderMultilineRecord(ts, record, component, message, orderedKeys, attrs)
	}
	return h.renderSingleLineRecord(ts, record, component, message, orderedKeys, attrs)
}

func (h *Handler) renderSingleLineRecord(ts time.Time, record slog.Record, component, message string, orderedKeys []string, attrs map[string]slog.Value) strings.Builder {
	var line strings.Builder
	line.WriteString(h.styles.Timestamp.Render(ts.Format(h.opts.TimeFormat)))
	line.WriteByte(' ')
	line.WriteString(h.renderLevel(record.Level))
	line.WriteByte(' ')
	line.WriteString(h.styles.Component.Render(padOrTrim(component, h.opts.ComponentWidth)))
	if h.opts.AddSource {
		if source := sourceFromPC(record.PC); source != "" {
			line.WriteByte(' ')
			line.WriteString(h.styles.Source.Render(source))
		}
	}
	line.WriteByte(' ')
	line.WriteString(h.styles.Message.Render(message))
	for _, key := range orderedKeys {
		line.WriteByte(' ')
		line.WriteString(h.styles.Key.Render(key))
		line.WriteByte('=')
		line.WriteString(h.renderValue(key, attrs[key]))
	}
	line.WriteByte('\n')
	return line
}

func (h *Handler) renderMultilineRecord(ts time.Time, record slog.Record, component, message string, orderedKeys []string, attrs map[string]slog.Value) strings.Builder {
	var line strings.Builder
	line.WriteString(h.styles.Timestamp.Render(ts.Format(h.opts.TimeFormat)))
	line.WriteByte(' ')
	line.WriteString(h.renderLevel(record.Level))
	line.WriteByte(' ')
	line.WriteString(h.styles.Component.Render(component))
	if h.opts.AddSource {
		if source := sourceFromPC(record.PC); source != "" {
			line.WriteByte(' ')
			line.WriteString(h.styles.Source.Render(source))
		}
	}
	line.WriteByte('\n')
	line.WriteString("  ")
	line.WriteString(h.styles.Key.Render("message"))
	line.WriteByte('=')
	line.WriteString(h.styles.Message.Render(quoteIfNeeded(message)))
	line.WriteByte('\n')
	for _, key := range orderedKeys {
		line.WriteString("  ")
		line.WriteString(h.styles.Key.Render(key))
		line.WriteByte('=')
		line.WriteString(h.renderValue(key, attrs[key]))
		line.WriteByte('\n')
	}
	if h.opts.Separator {
		line.WriteString(h.styles.Separator.Render(strings.Repeat("─", h.opts.SeparatorWidth)))
		line.WriteByte('\n')
	}
	return line
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	next := *h
	next.attrs = make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(next.attrs, h.attrs)
	copy(next.attrs[len(h.attrs):], attrs)
	return &next
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	next.groups = make([]string, len(h.groups)+1)
	copy(next.groups, h.groups)
	next.groups[len(h.groups)] = name
	return &next
}

func (h *Handler) appendAttr(dst map[string]slog.Value, groups []string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}
	if attr.Value.Kind() == slog.KindGroup {
		groupAttrs := attr.Value.Group()
		if len(groupAttrs) == 0 {
			return
		}
		nested := groups
		if attr.Key != "" {
			nested = append(copyStrings(groups), attr.Key)
		}
		for _, groupAttr := range groupAttrs {
			h.appendAttr(dst, nested, groupAttr)
		}
		return
	}
	key := joinAttrKey(groups, attr.Key)
	if key == "" {
		return
	}
	dst[key] = attr.Value
}

func (h *Handler) extractComponent(attrs map[string]slog.Value) string {
	raw, ok := attrs[h.opts.ComponentKey]
	if !ok {
		return "-"
	}
	delete(attrs, h.opts.ComponentKey)
	component := strings.TrimSpace(valueAsInlineString(raw))
	if component == "" {
		return "-"
	}
	return component
}

func (h *Handler) renderLevel(level slog.Level) string {
	switch {
	case level <= slog.LevelDebug:
		return h.styles.LevelDebug.Render("DBG")
	case level < slog.LevelWarn:
		return h.styles.LevelInfo.Render("INF")
	case level < slog.LevelError:
		return h.styles.LevelWarn.Render("WRN")
	default:
		return h.styles.LevelError.Render("ERR")
	}
}

func (h *Handler) renderValue(key string, value slog.Value) string {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return h.renderStringValue(key, value.String())
	case slog.KindDuration:
		return h.renderDuration(value.Duration())
	case slog.KindTime:
		return h.styles.Value.Render(value.Time().Format(time.RFC3339Nano))
	case slog.KindBool:
		return h.styles.Value.Render(strconv.FormatBool(value.Bool()))
	case slog.KindInt64:
		return h.renderIntegerValue(key, value.Int64())
	case slog.KindUint64:
		return h.renderUnsignedValue(key, value.Uint64())
	case slog.KindFloat64:
		return h.styles.Value.Render(strconv.FormatFloat(value.Float64(), 'f', -1, 64))
	case slog.KindAny:
		return h.renderAnyValue(key, value.Any())
	default:
		return h.styles.Value.Render(valueAsInlineString(value))
	}
}

func (h *Handler) renderStringValue(key, value string) string {
	clean := quoteIfNeeded(sanitizeOneLine(value))
	switch key {
	case "method":
		return h.methodStyle(value).Render(clean)
	case "request_id", "worker_id", "sync_task_id", "message_id", "job_id", "task_id", "user_id":
		return h.styles.IDValue.Render(clean)
	default:
		if isErrorKey(key) {
			return h.styles.ErrorValue.Render(clean)
		}
		return h.styles.Value.Render(clean)
	}
}

func (h *Handler) renderDuration(duration time.Duration) string {
	text := duration.String()
	switch {
	case duration < 100*time.Millisecond:
		return h.styles.DurationFast.Render(text)
	case duration < time.Second:
		return h.styles.DurationMedium.Render(text)
	default:
		return h.styles.DurationSlow.Render(text)
	}
}

func (h *Handler) renderIntegerValue(key string, value int64) string {
	text := strconv.FormatInt(value, 10)
	if key != "status" {
		return h.styles.Value.Render(text)
	}
	switch {
	case value >= 500:
		return h.styles.Status5xx.Render(text)
	case value >= 400:
		return h.styles.Status4xx.Render(text)
	case value >= 300:
		return h.styles.Status3xx.Render(text)
	default:
		return h.styles.Status2xx.Render(text)
	}
}

func (h *Handler) renderUnsignedValue(key string, value uint64) string {
	if value > uint64(mathMaxInt64) {
		return h.styles.Value.Render(strconv.FormatUint(value, 10))
	}
	return h.renderIntegerValue(key, int64(value))
}

func (h *Handler) renderAnyValue(key string, value any) string {
	switch typed := value.(type) {
	case nil:
		return h.styles.Value.Render("null")
	case error:
		return h.styles.ErrorValue.Render(quoteIfNeeded(sanitizeOneLine(typed.Error())))
	case time.Duration:
		return h.renderDuration(typed)
	case time.Time:
		return h.styles.Value.Render(typed.Format(time.RFC3339Nano))
	case fmt.Stringer:
		return h.renderStringValue(key, typed.String())
	case []byte:
		return h.renderStringValue(key, string(typed))
	default:
		if isErrorKey(key) {
			return h.styles.ErrorValue.Render(quoteIfNeeded(sanitizeOneLine(fmt.Sprint(typed))))
		}
		return h.styles.Value.Render(quoteIfNeeded(sanitizeOneLine(fmt.Sprint(typed))))
	}
}

func (h *Handler) methodStyle(method string) lipgloss.Style {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "HEAD", "OPTIONS":
		return h.styles.MethodRead
	case "POST", "PUT", "PATCH", "DELETE":
		return h.styles.MethodWrite
	default:
		return h.styles.MethodOther
	}
}

func defaultStyles(disableColor bool) Styles {
	if disableColor {
		plain := lipgloss.NewStyle()
		return Styles{
			Timestamp:      plain,
			LevelDebug:     plain,
			LevelInfo:      plain,
			LevelWarn:      plain,
			LevelError:     plain,
			Component:      plain,
			Source:         plain,
			Message:        plain,
			Key:            plain,
			Value:          plain,
			ErrorValue:     plain,
			IDValue:        plain,
			Status2xx:      plain,
			Status3xx:      plain,
			Status4xx:      plain,
			Status5xx:      plain,
			DurationFast:   plain,
			DurationMedium: plain,
			DurationSlow:   plain,
			MethodRead:     plain,
			MethodWrite:    plain,
			MethodOther:    plain,
			Separator:      plain,
		}
	}
	return Styles{
		Timestamp:      lipgloss.NewStyle().Faint(true),
		LevelDebug:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")),
		LevelInfo:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		LevelWarn:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")),
		LevelError:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("204")),
		Component:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")),
		Source:         lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("244")),
		Message:        lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		Key:            lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("244")),
		Value:          lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		ErrorValue:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("204")),
		IDValue:        lipgloss.NewStyle().Foreground(lipgloss.Color("45")),
		Status2xx:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")),
		Status3xx:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		Status4xx:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")),
		Status5xx:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203")),
		DurationFast:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		DurationMedium: lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		DurationSlow:   lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
		MethodRead:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")),
		MethodWrite:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213")),
		MethodOther:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("250")),
		Separator:      lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("242")),
	}
}

func orderedAttrKeys(attrs map[string]slog.Value) []string {
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}
	priority := map[string]int{
		"request_id":   0,
		"op":           1,
		"outcome":      2,
		"method":       3,
		"route":        4,
		"path":         5,
		"status":       6,
		"duration_ms":  7,
		"queue":        8,
		"worker_id":    9,
		"sync_task_id": 10,
		"message_id":   11,
		"error":        12,
	}
	sort.Slice(keys, func(i, j int) bool {
		pi, iok := priority[keys[i]]
		pj, jok := priority[keys[j]]
		switch {
		case iok && jok:
			if pi != pj {
				return pi < pj
			}
			return keys[i] < keys[j]
		case iok:
			return true
		case jok:
			return false
		default:
			return keys[i] < keys[j]
		}
	})
	return keys
}

func joinAttrKey(groups []string, key string) string {
	switch {
	case len(groups) == 0:
		return key
	case key == "":
		return strings.Join(groups, ".")
	default:
		return strings.Join(append(copyStrings(groups), key), ".")
	}
}

func copyStrings(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, len(input))
	copy(out, input)
	return out
}

func padOrTrim(value string, width int) string {
	if width <= 0 {
		return value
	}
	if len(value) == width {
		return value
	}
	if len(value) < width {
		return value + strings.Repeat(" ", width-len(value))
	}
	if width <= 1 {
		return value[:width]
	}
	return value[:width-1] + "…"
}

func sourceFromPC(pc uintptr) string {
	if pc == 0 {
		return ""
	}
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	if frame.File == "" || frame.Line <= 0 {
		return ""
	}
	return filepath.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
}

func valueAsInlineString(value slog.Value) string {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return sanitizeOneLine(value.String())
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindBool:
		return strconv.FormatBool(value.Bool())
	case slog.KindInt64:
		return strconv.FormatInt(value.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(value.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(value.Float64(), 'f', -1, 64)
	case slog.KindAny:
		return sanitizeOneLine(fmt.Sprint(value.Any()))
	default:
		return sanitizeOneLine(fmt.Sprint(value))
	}
}

func sanitizeOneLine(value string) string {
	value = strings.ReplaceAll(value, "\r", "\\r")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\t", "\\t")
	return value
}

func quoteIfNeeded(value string) string {
	if value == "" {
		return `""`
	}
	for _, r := range value {
		if unicode.IsSpace(r) || r == '=' || r == '"' {
			return strconv.Quote(value)
		}
	}
	return value
}

func isErrorKey(key string) bool {
	if strings.Contains(key, "stderr") {
		return false
	}
	return key == "error" ||
		key == "err" ||
		strings.HasPrefix(key, "error") ||
		strings.HasPrefix(key, "err") ||
		strings.HasSuffix(key, ".error") ||
		strings.HasSuffix(key, ".err")
}

const mathMaxInt64 = int64(^uint64(0) >> 1)
