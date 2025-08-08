package tui

import (
    "strings"
    "testing"
    "github.com/monster0506/mechexec/internal"
    "github.com/monster0506/mechexec/internal/logging"
)

func TestModel_InitAndPeerUpdate(t *testing.T) {
    m := newModel(logging.NewLogger("error"))
    if m.peerList.FilterState() == 0 { // ensure list created
        // ok
    }
    peers := []internal.PeerInfo{{Name: "alpha"}, {Name: "bravo"}}
    m2, _ := m.Update(peersUpdateMsg{Peers: peers})
    mm := m2.(model)
    if len(mm.peerList.Items()) != 2 {
        t.Fatalf("expected 2 peers, got %d", len(mm.peerList.Items()))
    }
}

func TestModel_ResultFiltering(t *testing.T) {
    m := newModel(logging.NewLogger("error"))
    res := &internal.ExecutionResults{
        Results: []internal.ExecutionResult{
            {Device: "alpha", Status: "ok", Stdout: "hello"},
            {Device: "bravo", Status: "failed", Stderr: "boom"},
        },
    }
    m.results = res
    m.tab = tabResults
    // type in filter
    m.resultFilter.SetValue("alpha")
    // render should include alpha and not bravo
    out := m.renderResults()
    if !contains(out, "alpha") || contains(out, "bravo") {
        t.Fatalf("filter not applied correctly: %s", out)
    }
}

func TestNewModelWithInitialView(t *testing.T) {
    mPeers := newModelWithInitialView(logging.NewLogger("error"), "peers")
    if mPeers.tab != tabPeers {
        t.Fatalf("expected peers tab, got %v", mPeers.tab)
    }
    mOverview := newModelWithInitialView(logging.NewLogger("error"), "overview")
    if mOverview.tab != tabPeers {
        t.Fatalf("expected overview->peers tab, got %v", mOverview.tab)
    }
    mResults := newModelWithInitialView(logging.NewLogger("error"), "results")
    if mResults.tab != tabResults {
        t.Fatalf("expected results tab, got %v", mResults.tab)
    }
    mCommands := newModelWithInitialView(logging.NewLogger("error"), "commands")
    if mCommands.tab != tabCommands {
        t.Fatalf("expected commands tab, got %v", mCommands.tab)
    }
    mUnknown := newModelWithInitialView(logging.NewLogger("error"), "nope")
    if mUnknown.tab != tabPeers {
        t.Fatalf("expected fallback to peers tab, got %v", mUnknown.tab)
    }
}

// helpers
func contains(s, sub string) bool { return strings.Contains(s, sub) }

