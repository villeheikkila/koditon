package cli

import (
	"encoding/json"
	"io"
)

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return string(raw)
	}
	compact, err := json.Marshal(value)
	if err != nil {
		return string(raw)
	}
	return string(compact)
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(resolveOutput(out))
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func WriteJSON(out io.Writer, value any) error {
	return writeJSON(out, value)
}

func resolveOutput(out io.Writer) io.Writer {
	if out == nil {
		return io.Discard
	}
	return out
}
