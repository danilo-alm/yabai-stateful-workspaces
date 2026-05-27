// Package state manages the mutable runtime state of the workspace daemon.
// It is safe for concurrent use via an internal mutex.
package state

import (
	"fmt"
	"sync"
)

// Manager holds and synchronises the daemon's mutable runtime state.
type Manager struct {
	mu            sync.RWMutex
	activeSetNum  int
}

// New returns a Manager initialised to set number 1.
func New() *Manager {
	return &Manager{activeSetNum: 1}
}

// SetNumber returns the current active set number.
func (m *Manager) SetNumber() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeSetNum
}

// UpdateSetNumber atomically changes the active set number.
// n must be in [1, 9]; any other value is silently ignored.
func (m *Manager) UpdateSetNumber(n int) error {
	if n < 1 || n > 9 {
		return fmt.Errorf("set number %d is out of valid range [1-9]", n)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeSetNum = n
	return nil
}

// DynamicSpaceName returns the space name for a dynamic key combined with
// the current set number, e.g. "q1", "a2".
func (m *Manager) DynamicSpaceName(letter byte) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return fmt.Sprintf("%c%d", letter, m.activeSetNum)
}
