package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hollis-labs/plugin"
)

// Host implements the plugin.Host interface for Nil.
// Nil is a Wails desktop app — it has no HTTP server, so CRUD and UI
// registration are no-ops that log warnings.
type Host struct {
	mu           sync.RWMutex
	plugins      map[string]plugin.Plugin
	eventHooks   map[string][]plugin.EventHook
	crudHandlers map[string]plugin.CRUDHandler
	services     map[string]interface{}
	configs      map[string]*PluginConfig
	activePlugin string
	logger       plugin.Logger
	ctx          context.Context
	ctxCancel    context.CancelFunc
}

// NewHost creates a new plugin host for Nil.
func NewHost(logger plugin.Logger) *Host {
	ctx, cancel := context.WithCancel(context.Background())
	return &Host{
		plugins:      make(map[string]plugin.Plugin),
		eventHooks:   make(map[string][]plugin.EventHook),
		crudHandlers: make(map[string]plugin.CRUDHandler),
		services:     make(map[string]interface{}),
		configs:      make(map[string]*PluginConfig),
		logger:       logger,
		ctx:          ctx,
		ctxCancel:    cancel,
	}
}

// NewHostMinimal creates a minimal plugin host for CLI operations.
func NewHostMinimal() *Host {
	return NewHost(NewLogger("plugin-cli"))
}

// GetPlugin retrieves another loaded plugin by ID.
func (h *Host) GetPlugin(id string) (plugin.Plugin, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	p, exists := h.plugins[id]
	return p, exists
}

// RegisterCRUDHandler logs a warning — Nil has no HTTP server for CRUD routes.
func (h *Host) RegisterCRUDHandler(resourceType string, handler plugin.CRUDHandler) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.crudHandlers[resourceType] = handler
	h.logger.Warn("CRUD handler registered but Nil has no HTTP server — not wired to routes", "resourceType", resourceType)
	return nil
}

// RegisterEventHook registers an event hook for specific event types.
func (h *Host) RegisterEventHook(eventTypes []string, hook plugin.EventHook) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, eventType := range eventTypes {
		h.eventHooks[eventType] = append(h.eventHooks[eventType], hook)
	}

	h.logger.Info("registered event hook", "eventTypes", eventTypes)
	return nil
}

// RegisterUIComponent is a no-op for Nil (Wails handles its own UI).
func (h *Host) RegisterUIComponent(component plugin.UIComponent) error {
	h.logger.Warn("RegisterUIComponent called but Nil uses Wails UI — ignoring", "id", component.ID)
	return nil
}

// GetService provides access to core services.
func (h *Host) GetService(name string) (interface{}, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	service, exists := h.services[name]
	if !exists {
		return nil, fmt.Errorf("service %q not found", name)
	}
	return service, nil
}

// RegisterService registers a core service for plugin access.
func (h *Host) RegisterService(name string, service interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.services[name] = service
	h.logger.Info("registered service", "name", name)
}

// SetConfig persists a configuration value for the calling plugin.
func (h *Host) SetConfig(key string, value string) error {
	h.logger.Warn("SetConfig not yet implemented in Nil", "key", key)
	return nil
}

// RegisterConfigSchema registers config field definitions for the calling plugin.
func (h *Host) RegisterConfigSchema(fields []plugin.ConfigFieldDef) error {
	h.logger.Warn("RegisterConfigSchema not yet implemented in Nil")
	return nil
}

// RegisterConnector registers a named connector.
func (h *Host) RegisterConnector(name string, connector plugin.Connector) error {
	h.logger.Warn("RegisterConnector not yet implemented in Nil", "name", name)
	return nil
}

// RegisterProvider registers a runtime LLM provider.
func (h *Host) RegisterProvider(name string, provider interface{}) error {
	h.logger.Warn("RegisterProvider not yet implemented in Nil", "name", name)
	return nil
}

// RegisterCLIAdapter registers a runtime CLI adapter.
func (h *Host) RegisterCLIAdapter(name string, adapter interface{}) error {
	h.logger.Warn("RegisterCLIAdapter not yet implemented in Nil", "name", name)
	return nil
}

// SlashCommandDef defines a slash command (app-specific, not in shared plugin package).
type SlashCommandDef struct {
	Name        string
	Description string
}

// UISlotEntry defines a UI slot entry (app-specific).
type UISlotEntry struct {
	Slot string
	Name string
}

// KeybindingDef defines a keyboard shortcut (app-specific).
type KeybindingDef struct {
	Key         string
	Description string
}

// RegisterCommand registers a slash command.
func (h *Host) RegisterCommand(cmd SlashCommandDef) error {
	h.logger.Warn("RegisterCommand not yet implemented in Nil", "name", cmd.Name)
	return nil
}

// RegisterSlot registers a UI slot entry.
func (h *Host) RegisterSlot(entry UISlotEntry) error {
	h.logger.Warn("RegisterSlot not yet implemented in Nil", "slot", entry.Slot)
	return nil
}

// RegisterKeybinding registers a keyboard shortcut.
func (h *Host) RegisterKeybinding(kb KeybindingDef) error {
	h.logger.Warn("RegisterKeybinding not yet implemented in Nil", "key", kb.Key)
	return nil
}

// Logger provides a logger instance for the plugin.
func (h *Host) Logger() plugin.Logger {
	return h.logger
}

// Context returns the plugin's execution context.
func (h *Host) Context() context.Context {
	return h.ctx
}

// SetPluginConfig stores configuration for a plugin.
func (h *Host) SetPluginConfig(pluginID string, cfg *PluginConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.configs[pluginID] = cfg
}

// GetConfig returns a configuration value for the currently-loading plugin.
func (h *Host) GetConfig(key string) (string, error) {
	h.mu.RLock()
	id := h.activePlugin
	cfg := h.configs[id]
	h.mu.RUnlock()

	if cfg == nil {
		return "", fmt.Errorf("no config loaded for plugin %q", id)
	}
	return cfg.Get(key)
}

// LoadPlugin loads a plugin into the host.
func (h *Host) LoadPlugin(p plugin.Plugin) error {
	id := p.ID()

	h.mu.Lock()
	if _, exists := h.plugins[id]; exists {
		h.mu.Unlock()
		return fmt.Errorf("plugin %q already loaded", id)
	}
	for _, dep := range p.Dependencies() {
		if _, exists := h.plugins[dep]; !exists {
			h.mu.Unlock()
			return fmt.Errorf("plugin %q depends on %q which is not loaded", id, dep)
		}
	}
	h.activePlugin = id
	h.mu.Unlock()

	if err := p.Load(h); err != nil {
		return fmt.Errorf("failed to load plugin %q: %w", id, err)
	}

	h.mu.Lock()
	h.plugins[id] = p
	h.mu.Unlock()

	h.logger.Info("loaded plugin", "id", id, "name", p.Name(), "version", p.Version())
	return nil
}

// UnloadPlugin unloads a plugin from the host.
func (h *Host) UnloadPlugin(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	p, exists := h.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %q not found", id)
	}

	for _, other := range h.plugins {
		if other.ID() == id {
			continue
		}
		for _, dep := range other.Dependencies() {
			if dep == id {
				return fmt.Errorf("cannot unload plugin %q: plugin %q depends on it", id, other.ID())
			}
		}
	}

	if err := p.Unload(); err != nil {
		return fmt.Errorf("failed to unload plugin %q: %w", id, err)
	}

	delete(h.plugins, id)
	h.logger.Info("unloaded plugin", "id", id)
	return nil
}

// EmitEvent emits an event to all registered hooks.
func (h *Host) EmitEvent(event plugin.Event) {
	h.mu.RLock()
	hooks, exists := h.eventHooks[event.Type]
	h.mu.RUnlock()

	if !exists {
		return
	}

	var wg sync.WaitGroup
	for _, hook := range hooks {
		wg.Add(1)
		go func(hook plugin.EventHook) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(h.ctx, 5*time.Second)
			defer cancel()

			if err := hook.Handle(ctx, event); err != nil {
				h.logger.Error("event hook failed", "eventType", event.Type, "error", err)
			}
		}(hook)
	}
	wg.Wait()
}

// ListPlugins returns all loaded plugins.
func (h *Host) ListPlugins() []plugin.Plugin {
	h.mu.RLock()
	defer h.mu.RUnlock()

	plugins := make([]plugin.Plugin, 0, len(h.plugins))
	for _, p := range h.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// Shutdown gracefully shuts down the plugin host and all loaded plugins.
func (h *Host) Shutdown() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var errors []string
	for id, p := range h.plugins {
		if err := p.Unload(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to unload plugin %q: %v", id, err))
		}
	}

	h.ctxCancel()

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	h.logger.Info("plugin host shutdown complete")
	return nil
}
