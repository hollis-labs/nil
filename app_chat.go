package main

import (
	"fmt"

	"github.com/hollis-labs/nil/chat"
	"github.com/hollis-labs/nil/config"
)

// --- Chat addon (F5) ---

// chatReady returns an error if the chat subsystem failed to initialize.
func (a *App) chatReady() error {
	if a.chatStore == nil || a.chatBridge == nil || a.chatRunner == nil {
		return fmt.Errorf("chat subsystem not initialized")
	}
	return nil
}

// StartChatSession opens a new chat session against the active vault.
func (a *App) StartChatSession() (*chat.ChatSession, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	dryRun := cfg.Chat.DryRun
	vaultID := ""
	if a.vaultMgr != nil {
		vaultID = a.vaultMgr.GetActiveVaultID()
	}
	return a.chatStore.CreateSession(a.ctx, vaultID, dryRun)
}

// getSessionCache returns (creating if needed) the in-memory tool cache for a session.
func (a *App) getSessionCache(sessionID int64) *chat.ToolCache {
	a.sessionCachesMu.Lock()
	defer a.sessionCachesMu.Unlock()
	if a.sessionCaches == nil {
		a.sessionCaches = make(map[int64]*chat.ToolCache)
	}
	if c, ok := a.sessionCaches[sessionID]; ok {
		return c
	}
	c := chat.NewToolCache(20)
	a.sessionCaches[sessionID] = c
	return c
}

// dropSessionCache frees the in-memory cache when a session ends.
func (a *App) dropSessionCache(sessionID int64) {
	a.sessionCachesMu.Lock()
	defer a.sessionCachesMu.Unlock()
	delete(a.sessionCaches, sessionID)
}

// EndChatSession closes the given session and frees its tool cache.
func (a *App) EndChatSession(sessionID int64) error {
	if err := a.chatReady(); err != nil {
		return err
	}
	a.dropSessionCache(sessionID)
	return a.chatStore.EndSession(a.ctx, sessionID)
}

// SendChatMessage sends a user message, calls the LLM, and returns the response.
// If the LLM proposes an action, the proposal is persisted and its ID returned.
func (a *App) SendChatMessage(sessionID int64, content string) (*chat.ChatResponse, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	// Persist the user message.
	userMsg, err := a.chatStore.AddMessage(a.ctx, &chat.ChatMessage{
		SessionID: sessionID,
		Role:      "user",
		Content:   content,
	})
	if err != nil {
		return nil, fmt.Errorf("chat: store user message: %w", err)
	}

	// Load session for context.
	sess, err := a.chatStore.GetSession(a.ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Get active vault metadata.
	vaultName := "Default"
	caps := chat.VaultCaps{Read: true}
	if a.vaultMgr != nil {
		if v := a.vaultMgr.GetActiveVault(); v != nil {
			vaultName = v.Name
			if vc, ok := cfg.Chat.VaultCaps[v.ID]; ok {
				caps = chat.VaultCaps{Read: true, Write: vc.Write, Delete: vc.Delete, DirectCreate: vc.DirectCreate}
			}
		}
	}

	// Load message history for context window.
	history, _ := a.chatStore.GetMessages(a.ctx, sessionID)
	_ = userMsg // already in history

	// Determine the active store for tool execution.
	var activeStore chat.BridgeStore
	if a.vaultMgr != nil {
		if s := a.vaultMgr.ActiveStore(); s != nil {
			activeStore = s
		}
	}

	// Call the LLM (agentic tool-use loop inside Bridge.Send).
	resp, err := a.chatBridge.Send(a.ctx, chat.BridgeRequest{
		APIKey:      cfg.Chat.APIKey,
		Model:       cfg.Chat.Model,
		VaultID:     sess.VaultID,
		VaultName:   vaultName,
		Caps:        caps,
		DryRun:      sess.DryRun,
		History:     history,
		UserMessage: content,
		Store:       activeStore,
		Templates:   a.chatStore,
		ToolCache:   a.getSessionCache(sessionID),
	})
	if err != nil {
		return nil, err
	}

	// Persist the assistant message.
	resp.Message.SessionID = sessionID
	assistantMsg, err := a.chatStore.AddMessage(a.ctx, &resp.Message)
	if err != nil {
		return nil, fmt.Errorf("chat: store assistant message: %w", err)
	}
	resp.Message = *assistantMsg

	// Persist tool calls made during this turn (best-effort; non-fatal on error).
	for _, tc := range resp.ToolCalls {
		_ = a.chatStore.PersistToolCall(a.ctx, sessionID, assistantMsg.ID, tc)
	}

	// If the LLM proposed an action, persist it.
	if resp.Proposal != nil {
		resp.Proposal.SessionID = sessionID
		resp.Proposal.MessageID = &assistantMsg.ID
		proposalID, err := a.chatRunner.Propose(a.ctx, resp.Proposal)
		if err != nil {
			return resp, fmt.Errorf("chat: persist proposal: %w", err)
		}
		resp.ProposalID = &proposalID
		// Reload full proposal with DB-assigned fields.
		full, _ := a.chatStore.GetProposal(a.ctx, proposalID)
		resp.Proposal = full
	}

	return resp, nil
}

// ApproveChatAction executes an approved ActionProposal.
func (a *App) ApproveChatAction(proposalID int64) (*chat.ActionResult, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	p, err := a.chatStore.GetProposal(a.ctx, proposalID)
	if err != nil {
		return nil, err
	}

	caps := chat.VaultCaps{Read: true}
	if vc, ok := cfg.Chat.VaultCaps[p.VaultID]; ok {
		caps = chat.VaultCaps{Read: true, Write: vc.Write, Delete: vc.Delete, DirectCreate: vc.DirectCreate}
	}

	return a.chatRunner.Approve(a.ctx, proposalID, a.vaultMgr.ActiveStore(), cfg.Chat.DryRun, caps)
}

// DenyChatAction rejects a pending ActionProposal.
func (a *App) DenyChatAction(proposalID int64) error {
	if err := a.chatReady(); err != nil {
		return err
	}
	return a.chatRunner.Deny(a.ctx, proposalID)
}

// GetChatHistory returns all messages for a session.
func (a *App) GetChatHistory(sessionID int64) ([]chat.ChatMessage, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	return a.chatStore.GetMessages(a.ctx, sessionID)
}

// GetActionAudit returns the most recent audit entries.
func (a *App) GetActionAudit(limit int) ([]chat.AuditEntry, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	return a.chatStore.GetAudit(a.ctx, limit)
}

// GetChatConfig returns the current chat configuration.
func (a *App) GetChatConfig() (*config.ChatConfig, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return &cfg.Chat, nil
}

// SetChatConfig persists updated chat configuration.
func (a *App) SetChatConfig(chatCfg config.ChatConfig) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	cfg.Chat = chatCfg
	// Reload bridge with new API key/model on next send (stateless Bridge, no action needed).
	return config.Save(cfg)
}

// ListChatTemplates returns all templates stored in chat.db.
func (a *App) ListChatTemplates() ([]chat.Template, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	tmpl, err := a.chatStore.ListTemplates(a.ctx)
	if tmpl == nil {
		tmpl = []chat.Template{}
	}
	return tmpl, err
}

// SaveChatTemplate creates or updates a template.
// If a template with the same slug already exists it is updated; otherwise it is created.
func (a *App) SaveChatTemplate(t chat.Template) (*chat.Template, error) {
	if err := a.chatReady(); err != nil {
		return nil, err
	}
	// Ensure defaults.
	if t.Parameters == "" {
		t.Parameters = "[]"
	}
	if t.OutputFormat == "" {
		t.OutputFormat = "markdown"
	}
	if t.Type == "" {
		t.Type = "generation"
	}
	// Upsert: check if slug exists.
	existing, _ := a.chatStore.GetTemplate(a.ctx, t.Slug)
	if existing != nil {
		if err := a.chatStore.UpdateTemplate(a.ctx, &t); err != nil {
			return nil, err
		}
		return a.chatStore.GetTemplate(a.ctx, t.Slug)
	}
	return a.chatStore.CreateTemplate(a.ctx, &t)
}

// DeleteChatTemplate removes a template by slug.
func (a *App) DeleteChatTemplate(slug string) error {
	if err := a.chatReady(); err != nil {
		return err
	}
	return a.chatStore.DeleteTemplate(a.ctx, slug)
}
