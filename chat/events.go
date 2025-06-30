// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chat

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ChatEventHandler handles chat-related events and integrates with the message storage middleware
type ChatEventHandler struct {
	device     *store.Device
	middleware *MessageStorageMiddleware
}

// NewChatEventHandler creates a new chat event handler
func NewChatEventHandler(device *store.Device) *ChatEventHandler {
	return &ChatEventHandler{
		device:     device,
		middleware: NewMessageStorageMiddleware(device),
	}
}

// HandleEvent handles all chat-related events
func (h *ChatEventHandler) HandleEvent(evt interface{}) bool {
	ctx := context.Background()

	switch event := evt.(type) {
	case *events.Message:
		return h.handleMessage(ctx, event)
	case *events.Receipt:
		return h.handleReceipt(ctx, event)
	case *events.Connected:
		return h.handleConnected(ctx, event)
	case *events.Disconnected:
		return h.handleDisconnected(ctx, event)
	default:
		// Ignore other events
		return true
	}
}

// handleMessage handles message events
func (h *ChatEventHandler) handleMessage(ctx context.Context, evt *events.Message) bool {
	// Skip protocol messages and other system messages
	if evt.Info.Category == "peer" || evt.Info.Category == "system" {
		return true
	}

	// Skip messages from ourselves if they're not from our own device
	if evt.Info.IsFromMe && evt.Info.Sender != h.device.GetJID() {
		return true
	}

	// Handle different types of messages
	if evt.IsEdit {
		err := h.middleware.HandleMessageEdit(ctx, evt)
		if err != nil {
			h.device.Log.Errorf("Failed to handle message edit: %v", err)
			return false
		}
	} else if evt.Info.Edit == types.EditAttributeSenderRevoke || evt.Info.Edit == types.EditAttributeAdminRevoke {
		err := h.middleware.HandleMessageRevoke(ctx, evt)
		if err != nil {
			h.device.Log.Errorf("Failed to handle message revoke: %v", err)
			return false
		}
	} else {
		// Regular message
		err := h.middleware.HandleMessage(ctx, evt)
		if err != nil {
			h.device.Log.Errorf("Failed to handle message: %v", err)
			return false
		}
	}

	return true
}

// handleReceipt handles receipt events (reactions, read receipts, etc.)
func (h *ChatEventHandler) handleReceipt(ctx context.Context, evt *events.Receipt) bool {
	// Handle reactions (non-read, non-delivery receipts)
	if evt.Type != types.ReceiptTypeRead && evt.Type != types.ReceiptTypeDelivered {
		err := h.middleware.HandleReaction(ctx, evt)
		if err != nil {
			h.device.Log.Errorf("Failed to handle reaction: %v", err)
			return false
		}
	}

	// Update message status for read receipts
	if evt.Type == types.ReceiptTypeRead {
		err := h.updateMessageStatus(ctx, evt, types.MessageStatusRead)
		if err != nil {
			h.device.Log.Errorf("Failed to update message status: %v", err)
			return false
		}
	}

	// Update message status for delivery receipts
	if evt.Type == types.ReceiptTypeDelivered {
		err := h.updateMessageStatus(ctx, evt, types.MessageStatusDelivered)
		if err != nil {
			h.device.Log.Errorf("Failed to update message status: %v", err)
			return false
		}
	}

	return true
}

// handleConnected handles connection events
func (h *ChatEventHandler) handleConnected(ctx context.Context, evt *events.Connected) bool {
	h.device.Log.Infof("Chat history system connected")
	return true
}

// handleDisconnected handles disconnection events
func (h *ChatEventHandler) handleDisconnected(ctx context.Context, evt *events.Disconnected) bool {
	h.device.Log.Infof("Chat history system disconnected")
	return true
}

// updateMessageStatus updates the status of messages based on receipts
func (h *ChatEventHandler) updateMessageStatus(ctx context.Context, receipt *events.Receipt, status types.MessageStatus) error {
	for _, messageID := range receipt.MessageIDs {
		// Get the message
		msg, err := h.device.ChatHistory.GetMessage(ctx, receipt.Chat, messageID)
		if err != nil {
			return fmt.Errorf("failed to get message %s: %w", messageID, err)
		}

		if msg == nil {
			// Message not found, skip
			continue
		}

		// Update status
		msg.Status = status

		// Update the message
		err = h.device.ChatHistory.UpdateMessage(ctx, msg)
		if err != nil {
			return fmt.Errorf("failed to update message %s: %w", messageID, err)
		}
	}

	return nil
}

// ChatHistoryUpdated event is emitted when chat history is updated
type ChatHistoryUpdated struct {
	ChatJID    types.JID
	MessageID  types.MessageID
	UpdateType string // "new", "edit", "revoke", "reaction"
	Timestamp  time.Time
}

// MessageEdited event is emitted when a message is edited
type MessageEdited struct {
	ChatJID       types.JID
	MessageID     types.MessageID
	OriginalText  string
	NewText       string
	EditTimestamp time.Time
}

// MessageRevoked event is emitted when a message is revoked
type MessageRevoked struct {
	ChatJID    types.JID
	MessageID  types.MessageID
	RevokeTime time.Time
	RevokedBy  types.JID
}

// ConversationArchived event is emitted when a conversation is archived
type ConversationArchived struct {
	ChatJID     types.JID
	IsArchived  bool
	ArchiveTime time.Time
}

// ConversationPinned event is emitted when a conversation is pinned
type ConversationPinned struct {
	ChatJID  types.JID
	IsPinned bool
	PinTime  time.Time
}

// ConversationMuted event is emitted when a conversation is muted
type ConversationMuted struct {
	ChatJID    types.JID
	IsMuted    bool
	MutedUntil *time.Time
	MuteTime   time.Time
}

// MessageReactionAdded event is emitted when a reaction is added to a message
type MessageReactionAdded struct {
	ChatJID      types.JID
	MessageID    types.MessageID
	SenderJID    types.JID
	Emoji        string
	ReactionTime time.Time
}

// MessageReactionRemoved event is emitted when a reaction is removed from a message
type MessageReactionRemoved struct {
	ChatJID     types.JID
	MessageID   types.MessageID
	SenderJID   types.JID
	RemovalTime time.Time
}
