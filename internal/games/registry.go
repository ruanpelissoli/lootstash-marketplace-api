package games

import (
	"fmt"
	"sync"
)

// Registry holds all registered game handlers
type Registry struct {
	handlers map[string]GameHandler
	mu       sync.RWMutex
}

var (
	defaultRegistry *Registry
	once            sync.Once
)

// GetRegistry returns the default game registry
func GetRegistry() *Registry {
	once.Do(func() {
		defaultRegistry = &Registry{
			handlers: make(map[string]GameHandler),
		}
	})
	return defaultRegistry
}

// Register adds a game handler to the registry
func (r *Registry) Register(handler GameHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[handler.GetCode()] = handler
}

// Get retrieves a game handler by code
func (r *Registry) Get(code string) (GameHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, ok := r.handlers[code]
	if !ok {
		return nil, fmt.Errorf("game not found: %s", code)
	}
	return handler, nil
}

// List returns all registered game handlers
func (r *Registry) List() []GameHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := make([]GameHandler, 0, len(r.handlers))
	for _, h := range r.handlers {
		handlers = append(handlers, h)
	}
	return handlers
}

// GetCategories returns categories for a specific game
func (r *Registry) GetCategories(code string) ([]Category, error) {
	handler, err := r.Get(code)
	if err != nil {
		return nil, err
	}
	return handler.GetCategories(), nil
}
