// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/util/dbutil"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// ChatHistoryStore implements store.ChatHistoryStore for SQL databases
type ChatHistoryStore struct {
	*SQLStore
}

// StoreMessage stores a message in the chat history
func (s *ChatHistoryStore) StoreMessage(ctx context.Context, msg *types.StoredMessage) error {
	// Serialize the message content
	messageContent, err := proto.Marshal(msg.Message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Extract search text from message
	searchText := s.extractSearchText(msg)

	// Prepare verified name details
	var verifiedNameDetails []byte
	if msg.VerifiedName != nil {
		verifiedNameDetails, err = proto.Marshal(msg.VerifiedName.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal verified name: %w", err)
		}
	}

	// Convert timestamps
	var editTimestamp *int64
	if msg.EditTimestamp != nil {
		ts := msg.EditTimestamp.Unix()
		editTimestamp = &ts
	}

	// Convert reply and thread IDs
	var replyToMessageID, replyToSenderJID, threadMessageID, threadSenderJID *string
	if msg.ReplyToMessageID != nil {
		id := string(*msg.ReplyToMessageID)
		replyToMessageID = &id
	}
	if msg.ReplyToSenderJID != nil {
		jid := msg.ReplyToSenderJID.String()
		replyToSenderJID = &jid
	}
	if msg.ThreadMessageID != nil {
		id := string(*msg.ThreadMessageID)
		threadMessageID = &id
	}
	if msg.ThreadSenderJID != nil {
		jid := msg.ThreadSenderJID.String()
		threadSenderJID = &jid
	}

	// Convert media information
	var mediaType, mediaID, mediaMimeType *string
	var mediaSize *int64
	if msg.MediaType != nil {
		mediaType = msg.MediaType
	}
	if msg.MediaID != nil {
		mediaID = msg.MediaID
	}
	if msg.MediaSize != nil {
		mediaSize = msg.MediaSize
	}
	if msg.MediaMimeType != nil {
		mediaMimeType = msg.MediaMimeType
	}

	// Convert additional metadata
	var pushName, category *string
	if msg.PushName != nil {
		pushName = msg.PushName
	}
	if msg.Category != nil {
		category = msg.Category
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO whatsmeow_chat_messages (
			our_jid, chat_jid, message_id, sender_jid, timestamp,
			message_content, message_type, is_from_me, is_group, is_edited, is_revoked, edit_timestamp,
			reply_to_message_id, reply_to_sender_jid, thread_message_id, thread_sender_jid,
			message_status, media_type, media_id, media_size, media_mime_type,
			push_name, verified_name_details, category, multicast, search_text
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
		ON CONFLICT (our_jid, chat_jid, message_id) DO UPDATE SET
			message_content = excluded.message_content,
			message_type = excluded.message_type,
			is_edited = excluded.is_edited,
			is_revoked = excluded.is_revoked,
			edit_timestamp = excluded.edit_timestamp,
			message_status = excluded.message_status,
			search_text = excluded.search_text
	`,
		s.JID, msg.ChatJID.String(), msg.ID, msg.SenderJID.String(), msg.Timestamp.Unix(),
		messageContent, msg.Type, msg.IsFromMe, msg.IsGroup, msg.IsEdited, msg.IsRevoked, editTimestamp,
		replyToMessageID, replyToSenderJID, threadMessageID, threadSenderJID,
		string(msg.Status), mediaType, mediaID, mediaSize, mediaMimeType,
		pushName, verifiedNameDetails, category, msg.Multicast, searchText,
	)

	return err
}

// GetMessages retrieves messages based on the query
func (s *ChatHistoryStore) GetMessages(ctx context.Context, query types.ChatHistoryQuery) ([]*types.StoredMessage, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Base conditions
	conditions = append(conditions, fmt.Sprintf("our_jid = $%d", argIndex))
	args = append(args, s.JID)
	argIndex++

	conditions = append(conditions, fmt.Sprintf("chat_jid = $%d", argIndex))
	args = append(args, query.ChatJID.String())
	argIndex++

	// Add optional filters
	if query.Before != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp < $%d", argIndex))
		args = append(args, query.Before.Unix())
		argIndex++
	}

	if query.After != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp > $%d", argIndex))
		args = append(args, query.After.Unix())
		argIndex++
	}

	if query.FromSender != nil {
		conditions = append(conditions, fmt.Sprintf("sender_jid = $%d", argIndex))
		args = append(args, query.FromSender.String())
		argIndex++
	}

	if query.MessageType != nil {
		conditions = append(conditions, fmt.Sprintf("message_type = $%d", argIndex))
		args = append(args, *query.MessageType)
		argIndex++
	}

	if query.SearchTerm != nil {
		conditions = append(conditions, fmt.Sprintf("search_text ILIKE $%d", argIndex))
		args = append(args, "%"+*query.SearchTerm+"%")
		argIndex++
	}

	// Build the query
	whereClause := strings.Join(conditions, " AND ")
	limitClause := ""
	if query.Limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, query.Limit)
	}

	sqlQuery := fmt.Sprintf(`
		SELECT message_id, sender_jid, timestamp, message_content, message_type,
		       is_from_me, is_group, is_edited, is_revoked, edit_timestamp,
		       reply_to_message_id, reply_to_sender_jid, thread_message_id, thread_sender_jid,
		       message_status, media_type, media_id, media_size, media_mime_type,
		       push_name, verified_name_details, category, multicast
		FROM whatsmeow_chat_messages
		WHERE %s
		ORDER BY timestamp DESC
		%s
	`, whereClause, limitClause)

	rows, err := s.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*types.StoredMessage
	for rows.Next() {
		msg, err := s.scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetMessage retrieves a specific message
func (s *ChatHistoryStore) GetMessage(ctx context.Context, chat types.JID, messageID types.MessageID) (*types.StoredMessage, error) {
	row := s.db.QueryRow(ctx, `
		SELECT message_id, sender_jid, timestamp, message_content, message_type,
		       is_from_me, is_group, is_edited, is_revoked, edit_timestamp,
		       reply_to_message_id, reply_to_sender_jid, thread_message_id, thread_sender_jid,
		       message_status, media_type, media_id, media_size, media_mime_type,
		       push_name, verified_name_details, category, multicast
		FROM whatsmeow_chat_messages
		WHERE our_jid = $1 AND chat_jid = $2 AND message_id = $3
	`, s.JID, chat.String(), messageID)

	return s.scanMessage(row)
}

// UpdateMessage updates an existing message
func (s *ChatHistoryStore) UpdateMessage(ctx context.Context, msg *types.StoredMessage) error {
	// For now, we'll just call StoreMessage which handles upserts
	return s.StoreMessage(ctx, msg)
}

// DeleteMessage deletes a message from the chat history
func (s *ChatHistoryStore) DeleteMessage(ctx context.Context, chat types.JID, messageID types.MessageID) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM whatsmeow_chat_messages
		WHERE our_jid = $1 AND chat_jid = $2 AND message_id = $3
	`, s.JID, chat.String(), messageID)

	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// GetMessageCount returns the number of messages in a chat
func (s *ChatHistoryStore) GetMessageCount(ctx context.Context, chat types.JID) (int64, error) {
	var count int64
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM whatsmeow_chat_messages
		WHERE our_jid = $1 AND chat_jid = $2
	`, s.JID, chat.String()).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to get message count: %w", err)
	}

	return count, nil
}

// GetLastMessage returns the most recent message in a chat
func (s *ChatHistoryStore) GetLastMessage(ctx context.Context, chat types.JID) (*types.StoredMessage, error) {
	row := s.db.QueryRow(ctx, `
		SELECT message_id, sender_jid, timestamp, message_content, message_type,
		       is_from_me, is_group, is_edited, is_revoked, edit_timestamp,
		       reply_to_message_id, reply_to_sender_jid, thread_message_id, thread_sender_jid,
		       message_status, media_type, media_id, media_size, media_mime_type,
		       push_name, verified_name_details, category, multicast
		FROM whatsmeow_chat_messages
		WHERE our_jid = $1 AND chat_jid = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`, s.JID, chat.String())

	return s.scanMessage(row)
}

// StoreReaction stores a message reaction
func (s *ChatHistoryStore) StoreReaction(ctx context.Context, reaction *types.MessageReaction) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO whatsmeow_message_reactions (our_jid, chat_jid, message_id, sender_jid, emoji, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (our_jid, chat_jid, message_id, sender_jid) DO UPDATE SET
			emoji = excluded.emoji,
			timestamp = excluded.timestamp
	`, s.JID, reaction.ChatJID.String(), reaction.MessageID, reaction.SenderJID.String(), reaction.Emoji, reaction.Timestamp.Unix())

	if err != nil {
		return fmt.Errorf("failed to store reaction: %w", err)
	}

	return nil
}

// GetReactions retrieves all reactions for a message
func (s *ChatHistoryStore) GetReactions(ctx context.Context, chat types.JID, messageID types.MessageID) ([]*types.MessageReaction, error) {
	rows, err := s.db.Query(ctx, `
		SELECT sender_jid, emoji, timestamp
		FROM whatsmeow_message_reactions
		WHERE our_jid = $1 AND chat_jid = $2 AND message_id = $3
	`, s.JID, chat.String(), messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reactions: %w", err)
	}
	defer rows.Close()

	var reactions []*types.MessageReaction
	for rows.Next() {
		var senderJIDStr, emoji string
		var timestamp int64
		err := rows.Scan(&senderJIDStr, &emoji, &timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reaction: %w", err)
		}

		senderJID, err := types.ParseJID(senderJIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sender JID: %w", err)
		}

		reactions = append(reactions, &types.MessageReaction{
			MessageID: messageID,
			ChatJID:   chat,
			SenderJID: senderJID,
			Emoji:     emoji,
			Timestamp: time.Unix(timestamp, 0),
		})
	}

	return reactions, nil
}

// DeleteReaction deletes a specific reaction
func (s *ChatHistoryStore) DeleteReaction(ctx context.Context, chat types.JID, messageID types.MessageID, sender types.JID) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM whatsmeow_message_reactions
		WHERE our_jid = $1 AND chat_jid = $2 AND message_id = $3 AND sender_jid = $4
	`, s.JID, chat.String(), messageID, sender.String())

	if err != nil {
		return fmt.Errorf("failed to delete reaction: %w", err)
	}

	return nil
}

// StoreForward stores message forwarding metadata
func (s *ChatHistoryStore) StoreForward(ctx context.Context, forward *types.MessageForward) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO whatsmeow_message_forwards (
			our_jid, chat_jid, message_id, original_message_id, original_chat_jid,
			original_sender_jid, original_timestamp, forward_timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (our_jid, chat_jid, message_id) DO UPDATE SET
			original_timestamp = excluded.original_timestamp,
			forward_timestamp = excluded.forward_timestamp
	`,
		s.JID, forward.OriginalChatJID.String(), forward.OriginalMessageID,
		forward.OriginalMessageID, forward.OriginalChatJID.String(),
		forward.OriginalSenderJID.String(), forward.OriginalTimestamp.Unix(), forward.ForwardTimestamp.Unix())

	if err != nil {
		return fmt.Errorf("failed to store forward: %w", err)
	}

	return nil
}

// GetForward retrieves forwarding metadata for a message
func (s *ChatHistoryStore) GetForward(ctx context.Context, chat types.JID, messageID types.MessageID) (*types.MessageForward, error) {
	row := s.db.QueryRow(ctx, `
		SELECT original_message_id, original_chat_jid, original_sender_jid, original_timestamp, forward_timestamp
		FROM whatsmeow_message_forwards
		WHERE our_jid = $1 AND chat_jid = $2 AND message_id = $3
	`, s.JID, chat.String(), messageID)

	var originalMessageID, originalChatJIDStr, originalSenderJIDStr string
	var originalTimestamp, forwardTimestamp int64

	err := row.Scan(&originalMessageID, &originalChatJIDStr, &originalSenderJIDStr, &originalTimestamp, &forwardTimestamp)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan forward: %w", err)
	}

	originalChatJID, err := types.ParseJID(originalChatJIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse original chat JID: %w", err)
	}

	originalSenderJID, err := types.ParseJID(originalSenderJIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse original sender JID: %w", err)
	}

	return &types.MessageForward{
		OriginalMessageID: types.MessageID(originalMessageID),
		OriginalChatJID:   originalChatJID,
		OriginalSenderJID: originalSenderJID,
		OriginalTimestamp: time.Unix(originalTimestamp, 0),
		ForwardTimestamp:  time.Unix(forwardTimestamp, 0),
	}, nil
}

// scanMessage scans a message from a database row
func (s *ChatHistoryStore) scanMessage(row dbutil.Scannable) (*types.StoredMessage, error) {
	var messageID, senderJIDStr, messageType, messageStatus string
	var timestamp, editTimestamp int64
	var messageContent []byte
	var isFromMe, isGroup, isEdited, isRevoked, multicast bool
	var replyToMessageID, replyToSenderJID, threadMessageID, threadSenderJID *string
	var mediaType, mediaID, mediaMimeType, pushName, category *string
	var mediaSize *int64
	var verifiedNameDetails []byte

	err := row.Scan(
		&messageID, &senderJIDStr, &timestamp, &messageContent, &messageType,
		&isFromMe, &isGroup, &isEdited, &isRevoked, &editTimestamp,
		&replyToMessageID, &replyToSenderJID, &threadMessageID, &threadSenderJID,
		&messageStatus, &mediaType, &mediaID, &mediaSize, &mediaMimeType,
		&pushName, &verifiedNameDetails, &category, &multicast,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan message: %w", err)
	}

	// Parse JID
	senderJID, err := types.ParseJID(senderJIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sender JID: %w", err)
	}

	// Unmarshal message content
	var message *waE2E.Message
	if len(messageContent) > 0 {
		msg := &waE2E.Message{}
		err := proto.Unmarshal(messageContent, msg)
		if err == nil {
			message = msg
		}
	}

	// Build the message
	msg := &types.StoredMessage{
		ID:        types.MessageID(messageID),
		SenderJID: senderJID,
		Timestamp: time.Unix(timestamp, 0),
		Message:   message,
		Type:      messageType,
		IsFromMe:  isFromMe,
		IsGroup:   isGroup,
		IsEdited:  isEdited,
		IsRevoked: isRevoked,
		Status:    types.MessageStatus(messageStatus),
		Multicast: multicast,
	}

	// Set optional fields
	if editTimestamp > 0 {
		ts := time.Unix(editTimestamp, 0)
		msg.EditTimestamp = &ts
	}

	if replyToMessageID != nil {
		id := types.MessageID(*replyToMessageID)
		msg.ReplyToMessageID = &id
	}

	if replyToSenderJID != nil {
		jid, err := types.ParseJID(*replyToSenderJID)
		if err == nil {
			msg.ReplyToSenderJID = &jid
		}
	}

	if threadMessageID != nil {
		id := types.MessageID(*threadMessageID)
		msg.ThreadMessageID = &id
	}

	if threadSenderJID != nil {
		jid, err := types.ParseJID(*threadSenderJID)
		if err == nil {
			msg.ThreadSenderJID = &jid
		}
	}

	// Set media fields
	msg.MediaType = mediaType
	msg.MediaID = mediaID
	msg.MediaSize = mediaSize
	msg.MediaMimeType = mediaMimeType

	// Set additional metadata
	msg.PushName = pushName
	msg.Category = category

	// Parse verified name if present
	if len(verifiedNameDetails) > 0 {
		// Unmarshal verified name details
		// This is a placeholder - implement based on your verified name structure
	}

	return msg, nil
}

// extractSearchText extracts searchable text from a message
func (s *ChatHistoryStore) extractSearchText(msg *types.StoredMessage) string {
	// This is a simplified implementation
	// You should extract text from the message content based on the message type
	var textParts []string

	// Add sender name if available
	if msg.PushName != nil && *msg.PushName != "" {
		textParts = append(textParts, *msg.PushName)
	}

	// Add message type
	textParts = append(textParts, msg.Type)

	// Add category if available
	if msg.Category != nil {
		textParts = append(textParts, *msg.Category)
	}

	// TODO: Extract actual message content based on message type
	// This would involve parsing the protobuf message and extracting text content

	return strings.Join(textParts, " ")
}
