package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// WriteJSON encodes v as indented JSON to w, followed by a newline.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteError writes a structured JSON error response to w.
func WriteError(w io.Writer, code, message, hint string) {
	resp := ErrorResponse{
		OK: false,
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Hint:    hint,
		},
	}
	_ = WriteJSON(w, resp)
}

// WriteNDJSON writes each item as a tagged NDJSON result line, then meta.
func WriteNDJSON(w io.Writer, items []ResultItem, meta NDJSONMeta) error {
	enc := json.NewEncoder(w)
	for _, item := range items {
		row := NDJSONResult{Type: "result", ResultItem: item}
		if err := enc.Encode(row); err != nil {
			return fmt.Errorf("encoding ndjson result: %w", err)
		}
	}
	return enc.Encode(meta)
}
