package choices

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectDuplicateJSONKeysAtEveryDepth(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload string
		valid   bool
	}{
		{"valid", `{"schema_version":1,"revision":0,"choices":{}}`, true},
		{"duplicate_root", `{"schema_version":1,"revision":0,"revision":1,"choices":{}}`, false},
		{"escaped_duplicate_root", `{"schema_version":1,"schem\u0061_version":1,"revision":0,"choices":{}}`, false},
		{"duplicate_purpose", `{"schema_version":1,"revision":0,"choices":{"demo-local-metrics":{},"demo-local-metrics":{}}}`, false},
		{"duplicate_nested_choice", `{"schema_version":1,"revision":1,"choices":{"demo-local-metrics":{"granted":true,"granted":false}}}`, false},
		{"reused_key_in_independent_objects", `[{"key":1},{"key":2}]`, true},
		{"trailing_object", `{}{}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := rejectDuplicateJSONKeys([]byte(test.payload))
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v, error=%v", test.valid, err)
			}
		})
	}
}

func TestLocalChoicesStoreRejectsAmbiguousJSONBeforePrivacyDecision(t *testing.T) {
	privateDir := t.TempDir()
	if err := os.Chmod(privateDir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(privateDir, "state.json")
	store := Store{Path: path}
	for _, payload := range []string{
		`{"schema_version":1,"revision":0,"revision":1,"choices":{}}`,
		`{"schema_version":1,"revision":1,"choices":{"demo-local-metrics":{"granted":true,"notice_version":"1","changed_at":"2026-11-07T00:00:00Z","revision":1},"demo-local-metrics":{"granted":false,"notice_version":"1","changed_at":"2026-11-07T00:00:00Z","revision":1}}}`,
		`{"schema_version":1,"revision":0,"choices":{},"private-secret-key":"foo","private-secret-key":"bar"}`,
	} {
		if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
			t.Fatal(err)
		}
		state, err := store.Read()
		if err == nil {
			t.Fatalf("duplicate JSON was accepted: state revision=%d", state.Revision)
		}
		if strings.Contains(err.Error(), "private-secret-key") || strings.Contains(err.Error(), "demo-local-metrics") {
			t.Fatal("private key name included in validation error")
		}
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":1,"revision":0,"choices":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	state, err := store.Read()
	if err != nil || state.SchemaVersion != 1 || state.Revision != 0 {
		t.Fatalf("valid local state was rejected: %+v, %v", state, err)
	}
}

func TestDuplicateKeyScannerRejectsExcessiveJSONNesting(t *testing.T) {
	// Test the boundary with independent nested arrays and objects.
	// Arbitrary depth must be rejected before recursive validator exhaustion.
	for _, tc := range []struct {
		name string
		valid string
		tooDeep string
	}{
		{"arrays", strings.Repeat("[", 64) + "0" + strings.Repeat("]", 64),
			strings.Repeat("[", 65) + "0" + strings.Repeat("]", 65)},
		{"objects", strings.Repeat("{\\\"k\\\":", 64) + "0" + strings.Repeat("}", 64),
			strings.Repeat("{\\\"k\\\":", 65) + "0" + strings.Repeat("}", 65)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := rejectDuplicateJSONKeys([]byte(tc.valid)); err != nil {
				t.Fatalf("valid nesting limit rejected: %v", err)
			}
			if err := rejectDuplicateJSONKeys([]byte(tc.tooDeep)); err == nil {
				t.Fatal("excessive nesting unexpectedly accepted")
			}
		})
	}
	if err := rejectDuplicateJSONKeys([]byte(strings.Repeat("[", 10_000))); err == nil {
		t.Fatal("unclosed adversarial nesting unexpectedly accepted")
	}
}
