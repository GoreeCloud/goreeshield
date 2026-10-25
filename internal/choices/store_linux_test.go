//go:build linux

package choices

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func storeForTest(t *testing.T) Store {
	t.Helper()
	return Store{Path: filepath.Join(t.TempDir(), "private", "choices.json")}
}

func TestStoreRoundTripPrivateFiles(t *testing.T) {
	st := storeForTest(t)
	initial, err := st.Read()
	if err != nil { t.Fatal(err) }
	if initial.Allowed("demo-local-metrics") { t.Fatal("first read allowed consent") }
	if err := st.Update(func(s *State) error { return s.Grant("demo-local-metrics", "1", true, fixed) }); err != nil { t.Fatal(err) }
	got, err := st.Read()
	if err != nil || !got.Allowed("demo-local-metrics") { t.Fatalf("readback: %+v, %v", got, err) }
	for _, path := range []string{st.Path, filepath.Join(filepath.Dir(st.Path), ".choices.lock")} {
		info, err := os.Stat(path)
		if err != nil { t.Fatal(err) }
		if info.Mode().Perm()&0077 != 0 { t.Fatalf("unsafe file permissions: %s %s", path, info.Mode()) }
	}
	if err := st.Update(func(s *State) error { return s.Withdraw("demo-local-metrics", fixed.Add(time.Minute)) }); err != nil { t.Fatal(err) }
	got, err = st.Read()
	if err != nil || got.Allowed("demo-local-metrics") { t.Fatal("revocation readback failed", err) }
}

func TestUntrustedOrCorruptStateFailsClosed(t *testing.T) {
	st := storeForTest(t)
	if _, err := st.Read(); err != nil { t.Fatal(err) }
	if err := os.WriteFile(st.Path, []byte(`{"schema_version":1,"choices":{},"unexpected":true}`), 0600); err != nil { t.Fatal(err) }
	if _, err := st.Read(); err == nil { t.Fatal("unknown fields accepted") }
	if err := os.WriteFile(st.Path, []byte(`{"schema_version":1,"choices":{}}`), 0644); err != nil { t.Fatal(err) }
	if err := os.Chmod(st.Path, 0644); err != nil { t.Fatal(err) }
	if _, err := st.Read(); err == nil { t.Fatal("insecure state permissions accepted") }
}

func TestSymlinkStateRejected(t *testing.T) {
	st := storeForTest(t)
	if _, err := st.Read(); err != nil { t.Fatal(err) }
	target := filepath.Join(t.TempDir(), "not-choice")
	if err := os.WriteFile(target, []byte("do not read"), 0600); err != nil { t.Fatal(err) }
	if err := os.Symlink(target, st.Path); err != nil { t.Fatal(err) }
	if _, err := st.Read(); err == nil { t.Fatal("state symlink should not be followed") }
}

func TestConcurrentUpdateIsSerialized(t *testing.T) {
	st := storeForTest(t)
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := st.Update(func(s *State) error {
				if i%2 == 0 { return s.Grant("demo-local-metrics", "1", true, fixed.Add(time.Duration(i)*time.Minute)) }
				return s.Withdraw("demo-local-metrics", fixed.Add(time.Duration(i)*time.Minute))
			})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs { if err != nil { t.Fatal(err) } }
	got, err := st.Read()
	if err != nil { t.Fatal(err) }
	if err := got.Validate(); err != nil { t.Fatal(err) }
}
