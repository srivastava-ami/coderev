package plugin

import "sync"

type Registry struct {
	mu      sync.Mutex
	plugins map[string]DiscoveredPlugin
}

func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]DiscoveredPlugin)}
}

func (r *Registry) Register(p DiscoveredPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[p.Manifest.Name] = p
}

func (r *Registry) Get(name string) (DiscoveredPlugin, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.plugins[name]
	return p, ok
}

func (r *Registry) List() []DiscoveredPlugin {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]DiscoveredPlugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		out = append(out, p)
	}
	return out
}
