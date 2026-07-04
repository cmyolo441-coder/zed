// Package plugin provides a plugin system for ZED. Users can add custom tools
// by implementing the Plugin interface and registering them. Plugins are
// hot-loadable — they can be added or removed at runtime without restarting.
package plugin

import (
	"context"
	"fmt"
	"sync"
)

// Plugin is a custom tool that extends ZED's capabilities.
type Plugin interface {
	// Name is the unique tool name the LLM uses to call this plugin.
	Name() string
	// Description tells the LLM what the plugin does.
	Description() string
	// Schema returns the JSON Schema for the plugin's arguments.
	Schema() map[string]any
	// Execute runs the plugin with raw JSON args.
	Execute(ctx context.Context, args string) (string, error)
}

// Manager holds all registered plugins and allows hot-loading.
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

// New creates an empty plugin manager.
func New() *Manager {
	return &Manager{plugins: map[string]Plugin{}}
}

// Register adds or replaces a plugin.
func (m *Manager) Register(p Plugin) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plugins[p.Name()] = p
}

// Unregister removes a plugin by name.
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.plugins, name)
}

// Get returns a plugin by name.
func (m *Manager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

// List returns all registered plugin names.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}

// Execute runs a plugin by name.
func (m *Manager) Execute(ctx context.Context, name, args string) (string, error) {
	p, ok := m.Get(name)
	if !ok {
		return "", fmt.Errorf("plugin %q not found", name)
	}
	return p.Execute(ctx, args)
}

// Count returns the number of registered plugins.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.plugins)
}
