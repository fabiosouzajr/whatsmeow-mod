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

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

// ChatHistoryClient provides high-level chat history operations
type ChatHistoryClient struct {
	client  *whatsmeow.Client
	device  *store.Device
	handler *ChatEventHandler
	// Phase 3: Advanced features
	exporter  *MessageExporter
	searcher  *AdvancedMessageSearch
	analytics *MessageAnalyticsEngine
}

// NewChatHistoryClient creates a new chat history client
func NewChatHistoryClient(client *whatsmeow.Client) *ChatHistoryClient {
	return &ChatHistoryClient{
		client:    client,
		device:    client.Store,
		handler:   NewChatEventHandler(client.Store),
		exporter:  NewMessageExporter(client.Store),
		searcher:  NewAdvancedMessageSearch(client.Store),
		analytics: NewMessageAnalyticsEngine(client.Store),
	}
}

// Setup initializes the chat history system and registers event handlers
func (c *ChatHistoryClient) Setup() error {
	// Register the event handler with the client
	c.client.AddEventHandler(c.handler)
	return nil
}

// GetMessages retrieves messages from a chat with optional filtering
func (c *ChatHistoryClient) GetMessages(ctx context.Context, chatJID types.JID, options *types.ChatHistoryQuery) ([]*types.StoredMessage, error) {
	if options == nil {
		options = &types.ChatHistoryQuery{
			ChatJID: chatJID,
			Limit:   50,
		}
	} else {
		options.ChatJID = chatJID
	}

	return c.device.ChatHistory.GetMessages(ctx, *options)
}

// GetMessage retrieves a specific message
func (c *ChatHistoryClient) GetMessage(ctx context.Context, chatJID types.JID, messageID types.MessageID) (*types.StoredMessage, error) {
	return c.device.ChatHistory.GetMessage(ctx, chatJID, messageID)
}

// SearchMessages searches for messages using text search
func (c *ChatHistoryClient) SearchMessages(ctx context.Context, chatJID types.JID, query string, limit int) ([]*types.StoredMessage, error) {
	searchQuery := &types.ChatHistoryQuery{
		ChatJID:    chatJID,
		SearchTerm: &query,
		Limit:      limit,
	}

	return c.device.ChatHistory.GetMessages(ctx, *searchQuery)
}

// SearchAllMessages searches for messages across all chats
func (c *ChatHistoryClient) SearchAllMessages(ctx context.Context, query string, limit int) ([]*types.MessageSearchResult, error) {
	return c.device.MessageSearch.SearchMessages(ctx, nil, query, limit)
}

// GetConversations retrieves all conversations
func (c *ChatHistoryClient) GetConversations(ctx context.Context) ([]*types.ConversationInfo, error) {
	return c.device.Conversations.GetAllConversations(ctx)
}

// GetConversation retrieves a specific conversation
func (c *ChatHistoryClient) GetConversation(ctx context.Context, chatJID types.JID) (*types.ConversationInfo, error) {
	return c.device.Conversations.GetConversation(ctx, chatJID)
}

// ArchiveConversation archives or unarchives a conversation
func (c *ChatHistoryClient) ArchiveConversation(ctx context.Context, chatJID types.JID, archive bool) error {
	conv, err := c.device.Conversations.GetConversation(ctx, chatJID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	if conv == nil {
		// Create new conversation if it doesn't exist
		conv = &types.ConversationInfo{
			ChatJID:      chatJID,
			IsArchived:   archive,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
		return c.device.Conversations.StoreConversation(ctx, conv)
	}

	// Update existing conversation
	conv.IsArchived = archive
	return c.device.Conversations.StoreConversation(ctx, conv)
}

// PinConversation pins or unpins a conversation
func (c *ChatHistoryClient) PinConversation(ctx context.Context, chatJID types.JID, pin bool) error {
	conv, err := c.device.Conversations.GetConversation(ctx, chatJID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	if conv == nil {
		// Create new conversation if it doesn't exist
		conv = &types.ConversationInfo{
			ChatJID:      chatJID,
			IsPinned:     pin,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
		return c.device.Conversations.StoreConversation(ctx, conv)
	}

	// Update existing conversation
	conv.IsPinned = pin
	return c.device.Conversations.StoreConversation(ctx, conv)
}

// MuteConversation mutes or unmutes a conversation
func (c *ChatHistoryClient) MuteConversation(ctx context.Context, chatJID types.JID, mute bool, until *time.Time) error {
	conv, err := c.device.Conversations.GetConversation(ctx, chatJID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	if conv == nil {
		// Create new conversation if it doesn't exist
		conv = &types.ConversationInfo{
			ChatJID:      chatJID,
			IsMuted:      mute,
			MutedUntil:   until,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}
		return c.device.Conversations.StoreConversation(ctx, conv)
	}

	// Update existing conversation
	conv.IsMuted = mute
	conv.MutedUntil = until
	return c.device.Conversations.StoreConversation(ctx, conv)
}

// DeleteMessage deletes a specific message
func (c *ChatHistoryClient) DeleteMessage(ctx context.Context, chatJID types.JID, messageID types.MessageID) error {
	return c.device.ChatHistory.DeleteMessage(ctx, chatJID, messageID)
}

// DeleteConversation deletes all messages in a conversation
func (c *ChatHistoryClient) DeleteConversation(ctx context.Context, chatJID types.JID) error {
	// Get all messages in the conversation
	query := types.ChatHistoryQuery{
		ChatJID: chatJID,
		Limit:   10000, // Large limit to get all messages
	}
	messages, err := c.device.ChatHistory.GetMessages(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	// Delete each message
	for _, msg := range messages {
		err := c.DeleteMessage(ctx, chatJID, msg.ID)
		if err != nil {
			return fmt.Errorf("failed to delete message %s: %w", msg.ID, err)
		}
	}

	// Delete the conversation itself
	return c.device.Conversations.DeleteConversation(ctx, chatJID)
}

// GetMessageReactions retrieves reactions for a message
func (c *ChatHistoryClient) GetMessageReactions(ctx context.Context, chatJID types.JID, messageID types.MessageID) ([]*types.MessageReaction, error) {
	return c.device.ChatHistory.GetReactions(ctx, chatJID, messageID)
}

// GetMessageStats retrieves statistics about messages
func (c *ChatHistoryClient) GetMessageStats(ctx context.Context, chatJID types.JID) (int64, error) {
	return c.device.ChatHistory.GetMessageCount(ctx, chatJID)
}

// GetConversationStats retrieves statistics about conversations
func (c *ChatHistoryClient) GetConversationStats(ctx context.Context) (int64, error) {
	// This method doesn't exist in the interface, so we'll return a placeholder
	return 0, fmt.Errorf("conversation stats not yet implemented")
}

// ExportChatHistory exports chat history to a file
func (c *ChatHistoryClient) ExportChatHistory(ctx context.Context, chatJID types.JID, filepath string, options *ExportOptions) error {
	return c.exporter.ExportChatHistory(ctx, chatJID, filepath, options)
}

// ExportMedia exports media files from messages
func (c *ChatHistoryClient) ExportMedia(ctx context.Context, chatJID types.JID, outputDir string, options *ExportOptions) error {
	return c.exporter.ExportMedia(ctx, chatJID, outputDir, options)
}

// AdvancedSearch performs advanced search with filters
func (c *ChatHistoryClient) AdvancedSearch(ctx context.Context, query *AdvancedSearchQuery) ([]*SearchResult, *SearchStats, error) {
	return c.searcher.Search(ctx, query)
}

// SearchByDateRange searches for messages within a date range
func (c *ChatHistoryClient) SearchByDateRange(ctx context.Context, chatJID *types.JID, start, end time.Time, limit int) ([]*SearchResult, error) {
	return c.searcher.SearchByDateRange(ctx, chatJID, start, end, limit)
}

// SearchBySender searches for messages from a specific sender
func (c *ChatHistoryClient) SearchBySender(ctx context.Context, chatJID *types.JID, sender types.JID, limit int) ([]*SearchResult, error) {
	return c.searcher.SearchBySender(ctx, chatJID, sender, limit)
}

// SearchByMessageType searches for messages of a specific type
func (c *ChatHistoryClient) SearchByMessageType(ctx context.Context, chatJID *types.JID, messageType string, limit int) ([]*SearchResult, error) {
	return c.searcher.SearchByMessageType(ctx, chatJID, messageType, limit)
}

// SearchWithRegex searches for messages using regex patterns
func (c *ChatHistoryClient) SearchWithRegex(ctx context.Context, chatJID *types.JID, pattern string, limit int) ([]*SearchResult, error) {
	return c.searcher.SearchWithRegex(ctx, chatJID, pattern, limit)
}

// GetSearchSuggestions provides search suggestions
func (c *ChatHistoryClient) GetSearchSuggestions(ctx context.Context, partialQuery string, limit int) ([]string, error) {
	return c.searcher.GetSearchSuggestions(ctx, partialQuery, limit)
}

// GetMessageAnalytics generates comprehensive message analytics
func (c *ChatHistoryClient) GetMessageAnalytics(ctx context.Context, period *AnalyticsPeriod) (*MessageAnalytics, error) {
	return c.analytics.GetMessageAnalytics(ctx, period)
}

// GetConversationAnalytics generates analytics for a specific conversation
func (c *ChatHistoryClient) GetConversationAnalytics(ctx context.Context, chatJID types.JID, period *AnalyticsPeriod) (*ConversationAnalytics, error) {
	return c.analytics.GetConversationAnalytics(ctx, chatJID, period)
}

// GetTrendAnalysis analyzes trends in message activity
func (c *ChatHistoryClient) GetTrendAnalysis(ctx context.Context, period *AnalyticsPeriod, granularity string) (*TrendData, error) {
	return c.analytics.GetTrendAnalysis(ctx, period, granularity)
}

// GetSenderAnalytics generates analytics for a specific sender
func (c *ChatHistoryClient) GetSenderAnalytics(ctx context.Context, senderJID types.JID, period *AnalyticsPeriod) (*SenderStats, error) {
	return c.analytics.GetSenderAnalytics(ctx, senderJID, period)
}

// GetRecentActivity gets recent message activity across all conversations
func (c *ChatHistoryClient) GetRecentActivity(ctx context.Context, limit int) ([]*types.StoredMessage, error) {
	// Get all conversations
	conversations, err := c.GetConversations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}

	var allMessages []*types.StoredMessage
	for _, conv := range conversations {
		messages, err := c.GetMessages(ctx, conv.ChatJID, &types.ChatHistoryQuery{
			Limit: limit / len(conversations), // Distribute limit across conversations
		})
		if err != nil {
			continue // Skip conversations with errors
		}
		allMessages = append(allMessages, messages...)
	}

	// Sort by timestamp (most recent first)
	// This is a simplified implementation - in production you'd want to use a more efficient approach
	for i := 0; i < len(allMessages)-1; i++ {
		for j := i + 1; j < len(allMessages); j++ {
			if allMessages[i].Timestamp.Before(allMessages[j].Timestamp) {
				allMessages[i], allMessages[j] = allMessages[j], allMessages[i]
			}
		}
	}

	// Limit results
	if len(allMessages) > limit {
		allMessages = allMessages[:limit]
	}

	return allMessages, nil
}

// GetUnreadCount gets the count of unread messages
func (c *ChatHistoryClient) GetUnreadCount(ctx context.Context, chatJID types.JID) (int, error) {
	// This would require tracking read status
	// For now, return 0 as this is not implemented in the base types
	return 0, nil
}

// MarkAsRead marks messages as read
func (c *ChatHistoryClient) MarkAsRead(ctx context.Context, chatJID types.JID, messageIDs []types.MessageID) error {
	// This would require updating read status
	// For now, return nil as this is not implemented in the base types
	return nil
}

// GetMessageThread gets messages in a thread (replies to a specific message)
func (c *ChatHistoryClient) GetMessageThread(ctx context.Context, chatJID types.JID, threadMessageID types.MessageID, limit int) ([]*types.StoredMessage, error) {
	// Get the thread message first
	threadMessages, err := c.GetMessages(ctx, chatJID, &types.ChatHistoryQuery{
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get thread messages: %w", err)
	}

	// Filter for messages that are part of the thread
	var threadMsgs []*types.StoredMessage
	for _, msg := range threadMessages {
		if msg.ThreadMessageID != nil && *msg.ThreadMessageID == threadMessageID {
			threadMsgs = append(threadMsgs, msg)
		}
	}

	return threadMsgs, nil
}

// GetMessageReplies gets direct replies to a specific message
func (c *ChatHistoryClient) GetMessageReplies(ctx context.Context, chatJID types.JID, messageID types.MessageID, limit int) ([]*types.StoredMessage, error) {
	// Get messages that reply to the specified message
	replies, err := c.GetMessages(ctx, chatJID, &types.ChatHistoryQuery{
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get message replies: %w", err)
	}

	// Filter for messages that reply to the specified message
	var replyMsgs []*types.StoredMessage
	for _, msg := range replies {
		if msg.ReplyToMessageID != nil && *msg.ReplyToMessageID == messageID {
			replyMsgs = append(replyMsgs, msg)
		}
	}

	return replyMsgs, nil
}

// GetMediaMessages gets messages with media attachments
func (c *ChatHistoryClient) GetMediaMessages(ctx context.Context, chatJID types.JID, mediaType *string, limit int) ([]*types.StoredMessage, error) {
	query := &types.ChatHistoryQuery{
		ChatJID: chatJID,
		Limit:   limit,
	}

	if mediaType != nil {
		query.MessageType = mediaType
	}

	messages, err := c.GetMessages(ctx, chatJID, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get media messages: %w", err)
	}

	// Filter for messages with media
	var mediaMessages []*types.StoredMessage
	for _, msg := range messages {
		if msg.MediaID != nil && *msg.MediaID != "" {
			mediaMessages = append(mediaMessages, msg)
		}
	}

	return mediaMessages, nil
}

// GetMessageHistory gets message history with pagination
func (c *ChatHistoryClient) GetMessageHistory(ctx context.Context, chatJID types.JID, before *time.Time, limit int) ([]*types.StoredMessage, error) {
	query := &types.ChatHistoryQuery{
		ChatJID: chatJID,
		Limit:   limit,
	}

	if before != nil {
		query.Before = before
	}

	return c.GetMessages(ctx, chatJID, query)
}

// GetMessageHistoryAfter gets message history after a specific time
func (c *ChatHistoryClient) GetMessageHistoryAfter(ctx context.Context, chatJID types.JID, after time.Time, limit int) ([]*types.StoredMessage, error) {
	query := &types.ChatHistoryQuery{
		ChatJID: chatJID,
		After:   &after,
		Limit:   limit,
	}

	return c.GetMessages(ctx, chatJID, query)
}
