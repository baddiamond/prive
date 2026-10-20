// Package engines holds the registry of messaging engines compiled into Privé.
//
// Each engine lives in its own sub-package and registers itself from an init function:
//
//	func init() { engines.Register(New()) }
//
// Bundled engines are enabled by blank imports in package all. See docs/ADDING_AN_ENGINE.md.
package engines

import (
	"fmt"
	"sort"
	"sync"

	"github.com/baddiamond/prive/internal/core"
)

// Registry is a set of engines indexed by manifest ID. It implements core.EngineSource.
type Registry struct {
	mu      sync.RWMutex
	engines map[string]core.Engine
}

// NewRegistry returns an empty registry. Most code uses the default registry through
// Register and Default; tests create their own.
func NewRegistry() *Registry {
	return &Registry{engines: map[string]core.Engine{}}
}

// Register adds an engine. It returns an error if the manifest is invalid or the ID is taken.
func (r *Registry) Register(e core.Engine) error {
	m := e.Manifest()
	if err := m.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.engines[m.ID]; dup {
		return fmt.Errorf("engines: %q registered twice", m.ID)
	}
	r.engines[m.ID] = e
	return nil
}

// Engine returns the engine with the given ID.
func (r *Registry) Engine(id string) (core.Engine, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.engines[id]
	return e, ok
}

// Manifests returns the manifests of all registered engines, sorted by ID.
func (r *Registry) Manifests() []core.Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]core.Manifest, 0, len(r.engines))
	for _, e := range r.engines {
		out = append(out, e.Manifest())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

var defaultRegistry = NewRegistry()

// Register adds an engine to the default registry. It panics on an invalid manifest or a
// duplicate ID, since both are programming errors caught at startup.
func Register(e core.Engine) {
	if err := defaultRegistry.Register(e); err != nil {
		panic(err)
	}
}

// Default returns the registry that bundled engines register into.
func Default() *Registry {
	return defaultRegistry
}
