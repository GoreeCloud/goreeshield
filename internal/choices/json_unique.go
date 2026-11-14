package choices

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// rejectDuplicateJSONKeys rejects ambiguity at every object depth, including
// escaped spellings of the same key. Go's ordinary JSON decoder accepts
// duplicates and would otherwise silently replace earlier decisions.
// Never echo a user-supplied key in errors: even field names may be private.
func rejectDuplicateJSONKeys(data []byte) error {
	const maxNesting = 64 // local state records are much shallower
	decoder := json.NewDecoder(bytes.NewReader(data))
	var visit func(depth int) error
	visit = func(depth int) error {
		// The input is bounded in bytes, but it can still have thousands of
		// nested containers. Bound recursive validation work before stack growth.
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		// Count nested containers only, not the scalar leaf of the last
		// permitted container. Excessive depth is refused before descent.
		if depth > maxNesting {
			return errors.New("JSON nesting limit exceeded")
		}
		switch delim {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				field, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := field.(string)
				if !ok {
					return errors.New("invalid JSON object key")
				}
				if _, duplicate := seen[name]; duplicate {
					return errors.New("duplicate JSON object key")
				}
				seen[name] = struct{}{}
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
			end, err := decoder.Token()
			if err != nil {
				return err
			}
			if end != json.Delim('}') {
				return errors.New("invalid JSON object ending")
			}
		case '[':
			for decoder.More() {
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
			end, err := decoder.Token()
			if err != nil {
				return err
			}
			if end != json.Delim(']') {
				return errors.New("invalid JSON array ending")
			}
		default:
			return errors.New("invalid JSON value")
		}
		return nil
	}
	if err := visit(1); err != nil {
		return errors.New("invalid or ambiguous JSON state")
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("invalid or trailing JSON state")
	}
	return nil
}
