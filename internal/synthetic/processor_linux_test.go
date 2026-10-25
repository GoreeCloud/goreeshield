//go:build linux

package synthetic

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GoreeCloud/goreeshield/internal/choices"
	"github.com/GoreeCloud/goreeshield/internal/contract"
)

func TestRevisionFencesStaleGrantsAndIdempotentReplay(t *testing.T) {
	store := choices.Store{Path: filepath.Join(t.TempDir(), "private", "choices.json")}
	sink := &Sink{}
	p := &Processor{Store: store, Sink: sink}
	request := contract.Request{Version: contract.Version, OperationID: "op-001", PurposeID: "demo-local-metrics", NoticeVersion: "1", ChoiceRevision: 1}
	if result, err := p.Process(request); err != nil || result.Executed || result.Status != "denied" {
		t.Fatalf("default deny: %+v %v", result, err)
	}
	if err := store.Update(func(s *choices.State) error { return s.Grant("demo-local-metrics", "1", true, time.Now().UTC()) }); err != nil {
		t.Fatal(err)
	}
	retry, err := p.Process(request)
	if err != nil || retry.Executed || retry.Status != "duplicate" || retry.PriorStatus != "denied" {
		t.Fatalf("denied id must not turn into grant: %+v %v", retry, err)
	}
	request.OperationID = "op-002"
	applied, err := p.Process(request)
	if err != nil || !applied.Executed || applied.Status != "applied" || sink.Count("demo-local-metrics") != 1 {
		t.Fatalf("granted synthetic attempt: %+v %v", applied, err)
	}
	again, err := p.Process(request)
	if err != nil || again.Executed || again.Status != "duplicate" || again.PriorStatus != "applied" || sink.Count("demo-local-metrics") != 1 {
		t.Fatalf("idempotency failed: %+v %v", again, err)
	}
	changed := request
	changed.PurposeID = "demo-remote-assistance"
	if result, err := p.Process(changed); err != nil || result.Status != "conflict" || result.Executed {
		t.Fatalf("operation id collision failed closed: %+v %v", result, err)
	}
	if err := store.Update(func(s *choices.State) error { return s.Withdraw("demo-local-metrics", time.Now().UTC()) }); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *choices.State) error { return s.Grant("demo-local-metrics", "1", true, time.Now().UTC()) }); err != nil {
		t.Fatal(err)
	}
	stale := request
	stale.OperationID = "op-003"
	if result, err := p.Process(stale); err != nil || result.Executed || result.Status != "denied" {
		t.Fatalf("stale revision after regrant must deny: %+v %v", result, err)
	}
	current, err := store.Read()
	if err != nil { t.Fatal(err) }
	if current.Revision != 3 { t.Fatalf("want revision 3, got %d", current.Revision) }
	fresh := request
	fresh.OperationID = "op-004"
	fresh.ChoiceRevision = current.Revision
	if result, err := p.Process(fresh); err != nil || !result.Executed || result.Status != "applied" || sink.Count("demo-local-metrics") != 2 {
		t.Fatalf("fresh revision should apply once: %+v %v", result, err)
	}
}

func TestConcurrentIdenticalOperationsExecuteOnce(t *testing.T) {
	store := choices.Store{Path: filepath.Join(t.TempDir(), "private", "choices.json")}
	if err := store.Update(func(s *choices.State) error { return s.Grant("demo-local-metrics", "1", true, time.Now().UTC()) }); err != nil { t.Fatal(err) }
	p := &Processor{Store: store, Sink: &Sink{}}
	req := contract.Request{Version: contract.Version, OperationID: "op-race", PurposeID: "demo-local-metrics", NoticeVersion: "1", ChoiceRevision: 1}
	var wg sync.WaitGroup
	resultCh := make(chan contract.Result, 16)
	errCh := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := p.Process(req)
			resultCh <- r
			errCh <- err
		}()
	}
	wg.Wait()
	close(resultCh)
	close(errCh)
	for err := range errCh { if err != nil { t.Fatal(err) } }
	executions := 0
	for result := range resultCh { if result.Executed { executions++ } }
	if executions != 1 || p.Sink.Count("demo-local-metrics") != 1 {
		t.Fatalf("concurrent duplicate applied %d times, sink %d", executions, p.Sink.Count("demo-local-metrics"))
	}
}

func TestInvalidEnvelopeAndSourceStateCannotExecute(t *testing.T) {
	store := choices.Store{Path: filepath.Join(t.TempDir(), "private", "choices.json")}
	p := &Processor{Store: store, Sink: &Sink{}}
	request := contract.Request{Version: contract.Version, OperationID: "op-good", PurposeID: "demo-local-metrics", NoticeVersion: "1", ChoiceRevision: 1}
	for _, bad := range []contract.Request{
		{Version: 2, OperationID: "op-good", PurposeID: "demo-local-metrics", NoticeVersion: "1", ChoiceRevision: 1},
		{Version: 1, OperationID: "USER@example.org", PurposeID: "demo-local-metrics", NoticeVersion: "1", ChoiceRevision: 1},
		{Version: 1, OperationID: "op-good", PurposeID: "demo-local-metrics", NoticeVersion: "0", ChoiceRevision: 1},
		{Version: 1, OperationID: "op-good", PurposeID: "demo-local-metrics", NoticeVersion: "1", ChoiceRevision: 0},
	} {
		if result, err := p.Process(bad); err == nil || result.Executed { t.Fatalf("invalid request accepted: %+v %v", bad, err) }
	}
	if result, err := p.Process(request); err != nil || result.Executed || result.Status != "denied" {
		t.Fatalf("missing state should deny: %+v %v", result, err)
	}
	if p.Sink.Count("demo-local-metrics") != 0 { t.Fatal("unexpected synthetic event") }
}
