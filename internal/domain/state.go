package domain

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

type Mode int

const (
	ModeLive Mode = iota
	ModeTest
)

type Manager struct {
	current Mode
	mu      sync.RWMutex
}

func NewModeManager() *Manager {
	return &Manager{
		current: ModeLive,
	}
}

func (m *Manager) SetMode(ctx context.Context, mode Mode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if mode != ModeLive && mode != ModeTest {
		return errors.New("invalid mode")
	}

	m.current = mode
	slog.Info("Mode switched", "mode", mode.String())
	return nil
}

func (m *Manager) GetMode() Mode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m Mode) String() string {
	switch m {
	case ModeLive:
		return "live"
	case ModeTest:
		return "test"
	default:
		return "unknown"
	}
}
