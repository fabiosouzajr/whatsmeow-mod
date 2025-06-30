// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package sqlstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
)

// MessageSearchStore implements store.MessageSearchStore for SQL databases
type MessageSearchStore struct {
	*SQLStore
}

// SearchMessages searches for messages using full-text search
func (s *MessageSearchStore) SearchMessages(ctx context.Context, query string, chat *types.JID, limit int) ([]*types.MessageSearchResult, error) {
	if limit <= 0 {
		limit = 50 // Default limit
	}

	var sqlQuery string
	var args []interface{}

	// Base query with full-text search
	if chat != nil {
		sqlQuery = `
			SELECT m.message_id, m.sender_jid, m.timestamp, m.message_content, m.message_type,
			       m.is_from_me, m.is_group, m.is_edited, m.is_revoked, m.edit_timestamp, m.message_status,
			       ts_rank(msi.search_vector, plainto_tsquery('english', $1)) as relevance
			FROM whatsmeow_chat_messages m
			JOIN whatsmeow_message_search_index msi ON m.our_jid = msi.our_jid AND m.chat_jid = msi.chat_jid AND m.message_id = msi.message_id
			WHERE m.our_jid = $2 AND m.chat_jid = $3 AND msi.search_vector @@ plainto_tsquery('english', $1)
			ORDER BY relevance DESC, m.timestamp DESC
			LIMIT $4
		`
		args = []interface{}{query, s.JID, chat.String(), limit}
	} else {
		sqlQuery = `
			SELECT m.message_id, m.sender_jid, m.timestamp, m.message_content, m.message_type,
			       m.is_from_me, m.is_group, m.is_edited, m.is_revoked, m.edit_timestamp, m.message_status,
			       ts_rank(msi.search_vector, plainto_tsquery('english', $1)) as relevance
			FROM whatsmeow_chat_messages m
			JOIN whatsmeow_message_search_index msi ON m.our_jid = msi.our_jid AND m.chat_jid = msi.chat_jid AND m.message_id = msi.message_id
			WHERE m.our_jid = $2 AND msi.search_vector @@ plainto_tsquery('english', $1)
			ORDER BY relevance DESC, m.timestamp DESC
			LIMIT $3
		`
		args = []interface{}{query, s.JID, limit}
	}

	rows, err := s.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	var results []*types.MessageSearchResult
	for rows.Next() {
		var messageID, senderJIDStr, messageType, messageStatus string
		var timestamp, editTimestamp int64
		var messageContent []byte
		var isFromMe, isGroup, isEdited, isRevoked bool
		var relevance float64

		err := rows.Scan(&messageID, &senderJIDStr, &timestamp, &messageContent, &messageType,
			&isFromMe, &isGroup, &isEdited, &isRevoked, &editTimestamp, &messageStatus, &relevance)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		senderJID, err := types.ParseJID(senderJIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sender JID: %w", err)
		}

		// Create stored message
		msg := &types.StoredMessage{
			ID:        types.MessageID(messageID),
			SenderJID: senderJID,
			Timestamp: time.Unix(timestamp, 0),
			Type:      messageType,
			IsFromMe:  isFromMe,
			IsGroup:   isGroup,
			IsEdited:  isEdited,
			IsRevoked: isRevoked,
			Status:    types.MessageStatus(messageStatus),
		}

		if editTimestamp > 0 {
			ts := time.Unix(editTimestamp, 0)
			msg.EditTimestamp = &ts
		}

		// Create search result
		result := &types.MessageSearchResult{
			Message:   msg,
			Relevance: relevance,
			Highlights: []string{
				fmt.Sprintf("Found in %s message", messageType),
			},
		}

		results = append(results, result)
	}

	return results, nil
}

// IndexMessage indexes a message for search
func (s *MessageSearchStore) IndexMessage(ctx context.Context, msg *types.StoredMessage) error {
	// Extract searchable text from the message
	searchText := s.extractSearchText(msg)

	// Update the search text in the message table
	_, err := s.db.Exec(ctx, `
		UPDATE whatsmeow_chat_messages
		SET search_text = $4
		WHERE our_jid = $1 AND chat_jid = $2 AND message_id = $3
	`, s.JID, msg.ChatJID.String(), msg.ID, searchText)

	if err != nil {
		return fmt.Errorf("failed to index message: %w", err)
	}

	return nil
}

// RemoveMessageFromIndex removes a message from the search index
func (s *MessageSearchStore) RemoveMessageFromIndex(ctx context.Context, chat types.JID, messageID types.MessageID) error {
	// Clear the search text
	_, err := s.db.Exec(ctx, `
		UPDATE whatsmeow_chat_messages
		SET search_text = NULL
		WHERE our_jid = $1 AND chat_jid = $2 AND message_id = $3
	`, s.JID, chat.String(), messageID)

	if err != nil {
		return fmt.Errorf("failed to remove message from index: %w", err)
	}

	return nil
}

// RebuildSearchIndex rebuilds the entire search index
func (s *MessageSearchStore) RebuildSearchIndex(ctx context.Context) error {
	// This is a simplified implementation
	// In a real implementation, you would:
	// 1. Clear all search text
	// 2. Re-extract search text from all messages
	// 3. Update the search index

	_, err := s.db.Exec(ctx, `
		UPDATE whatsmeow_chat_messages
		SET search_text = COALESCE(message_type, '') || ' ' || COALESCE(push_name, '')
		WHERE our_jid = $1 AND search_text IS NULL
	`, s.JID)

	if err != nil {
		return fmt.Errorf("failed to rebuild search index: %w", err)
	}

	return nil
}

// extractSearchText extracts searchable text from a message
func (s *MessageSearchStore) extractSearchText(msg *types.StoredMessage) string {
	var textParts []string

	// Add message type
	textParts = append(textParts, msg.Type)

	// Add sender name if available
	if msg.PushName != nil && *msg.PushName != "" {
		textParts = append(textParts, *msg.PushName)
	}

	// Add category if available
	if msg.Category != nil {
		textParts = append(textParts, *msg.Category)
	}

	// TODO: Extract actual message content based on message type
	// This would involve parsing the protobuf message and extracting text content
	// For now, we'll use a placeholder

	return strings.Join(textParts, " ")
}
