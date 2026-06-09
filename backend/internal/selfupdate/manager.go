package selfupdate

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Status is the coarse self-update state surfaced to the frontend.
type Status string

const (
	StatusIdle        Status = "idle"
	StatusChecking    Status = "checking"
	StatusDownloading Status = "downloading"
	StatusReady       Status = "ready"
	StatusError       Status = "error"
	StatusManual      Status = "manual"
)

// State is a snapshot of the manager for the status endpoint.
type State struct {
	Edition        string `json:"edition"`
	Status         Status `json:"status"`
	NeedsElevation bool   `json:"needsElevation"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseURL     string `json:"releaseUrl"`
	Progress       int    `json:"progress"` // download percent 0-100 while StatusDownloading
	Error          string `json:"error,omitempty"`
}

// Manager owns the desktop self-update lifecycle: periodic check, background
// download+verify, and applying a staged update on request.
type Manager struct {
	mu      sync.Mutex
	state   State
	staged  string
	current string

	// Seams (overridable in tests).
	fetchLatest func(context.Context) (tag, htmlURL string, err error)
	downloadFn  func(context.Context, string, func(int)) (string, error)
	writable    func() bool
	canElevate  func() bool
	applyFn     func(string) error
}

// NewManager builds a Manager wired to the real fetch/download/apply functions.
func NewManager(currentVersion string) *Manager {
	m := &Manager{
		current:     currentVersion,
		fetchLatest: fetchLatestRelease,
		downloadFn:  Download,
		writable:    Writable,
		canElevate:  canElevateFn,
		applyFn:     Apply,
	}
	m.state = State{
		Edition:        "desktop",
		Status:         StatusIdle,
		CurrentVersion: currentVersion,
	}
	return m
}

// Start runs an immediate check then re-checks every interval until ctx ends.
func (m *Manager) Start(ctx context.Context, interval time.Duration) {
	go func() {
		m.cycle(ctx)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.cycle(ctx)
			}
		}
	}()
}

func (m *Manager) setStatus(s Status, mutate func(*State)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.Status = s
	if mutate != nil {
		mutate(&m.state)
	}
}

// cycle performs one check→(download) pass. Once ready, it short-circuits so a
// staged update stays ready.
func (m *Manager) cycle(ctx context.Context) {
	m.mu.Lock()
	already := m.state.Status == StatusReady
	m.mu.Unlock()
	if already {
		return
	}

	m.setStatus(StatusChecking, nil)
	tag, htmlURL, err := m.fetchLatest(ctx)
	if err != nil {
		m.setStatus(StatusError, func(s *State) { s.Error = err.Error() })
		return
	}
	latest := strings.TrimPrefix(tag, "v")
	current := strings.TrimPrefix(m.current, "v")
	if current == "dev" || compareVersions(current, latest) >= 0 {
		m.setStatus(StatusIdle, func(s *State) {
			s.LatestVersion = latest
			s.ReleaseURL = htmlURL
			s.NeedsElevation = false
			s.Error = ""
		})
		return
	}

	writable := m.writable()
	if !writable && !m.canElevate() {
		m.setStatus(StatusManual, func(s *State) {
			s.LatestVersion = latest
			s.ReleaseURL = htmlURL
			s.NeedsElevation = false
			s.Error = ""
		})
		return
	}

	m.setStatus(StatusDownloading, func(s *State) {
		s.LatestVersion = latest
		s.ReleaseURL = htmlURL
		s.NeedsElevation = !writable
		s.Progress = 0
	})
	staged, err := m.downloadFn(ctx, tag, func(pct int) {
		m.mu.Lock()
		if m.state.Status == StatusDownloading {
			m.state.Progress = pct
		}
		m.mu.Unlock()
	})
	if err != nil {
		m.setStatus(StatusError, func(s *State) { s.Error = err.Error() })
		return
	}
	m.mu.Lock()
	m.staged = staged
	m.state.Status = StatusReady
	m.state.Progress = 100
	m.state.Error = ""
	m.mu.Unlock()
}

// Status returns a snapshot.
func (m *Manager) Status() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// CanRestart reports whether a verified update is staged and ready to apply.
func (m *Manager) CanRestart() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state.Status == StatusReady && m.staged != ""
}

// Restart applies a previously-staged, verified update and relaunches.
func (m *Manager) Restart() error {
	m.mu.Lock()
	ready := m.state.Status == StatusReady && m.staged != ""
	staged := m.staged
	m.mu.Unlock()
	if !ready {
		return fmt.Errorf("selfupdate: no staged update to apply")
	}
	if err := m.applyFn(staged); err != nil {
		m.mu.Lock()
		m.state.Status = StatusError
		m.state.Error = err.Error()
		m.mu.Unlock()
		return err
	}
	return nil
}
