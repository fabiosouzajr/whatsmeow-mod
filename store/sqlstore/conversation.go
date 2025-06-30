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
	"time"

	"go.mau.fi/whatsmeow/types"
)

// ConversationStore implements store.ConversationStore for SQL databases
type ConversationStore struct {
	*SQLStore
}

// StoreConversation stores conversation metadata
func (s *ConversationStore) StoreConversation(ctx context.Context, conv *types.ConversationInfo) error {
	var mutedUntil *int64
	if conv.MutedUntil != nil {
		ts := conv.MutedUntil.Unix()
		mutedUntil = &ts
	}

	var lastMessage *int64
	if conv.LastMessage != nil {
		ts := conv.LastMessage.Unix()
		lastMessage = &ts
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO whatsmeow_conversations (
			our_jid, chat_jid, name, description, is_group, is_broadcast,
			created_at, last_activity, last_message, is_archived, is_pinned,
			is_muted, muted_until, group_invite_link, message_count, unread_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (our_jid, chat_jid) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			last_activity = excluded.last_activity,
			last_message = excluded.last_message,
			is_archived = excluded.is_archived,
			is_pinned = excluded.is_pinned,
			is_muted = excluded.is_muted,
			muted_until = excluded.muted_until,
			group_invite_link = excluded.group_invite_link,
			message_count = excluded.message_count,
			unread_count = excluded.unread_count
	`,
		s.JID, conv.ChatJID.String(), conv.Name, conv.Description, conv.IsGroup, conv.IsBroadcast,
		conv.CreatedAt.Unix(), conv.LastActivity.Unix(), lastMessage, conv.IsArchived, conv.IsPinned,
		conv.IsMuted, mutedUntil, conv.GroupInviteLink, conv.MessageCount, conv.UnreadCount)

	if err != nil {
		return fmt.Errorf("failed to store conversation: %w", err)
	}

	// Store group participants if this is a group
	if conv.IsGroup && len(conv.GroupParticipants) > 0 {
		err = s.updateGroupParticipants(ctx, conv.ChatJID, conv.GroupParticipants, conv.GroupAdmins)
		if err != nil {
			return fmt.Errorf("failed to store group participants: %w", err)
		}
	}

	return nil
}

// GetConversation retrieves conversation metadata
func (s *ConversationStore) GetConversation(ctx context.Context, chat types.JID) (*types.ConversationInfo, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT name, description, is_group, is_broadcast, created_at, last_activity, last_message,
		       is_archived, is_pinned, is_muted, muted_until, group_invite_link, message_count, unread_count
		FROM whatsmeow_conversations
		WHERE our_jid = $1 AND chat_jid = $2
	`, s.JID, chat.String())

	var name, description *string
	var isGroup, isBroadcast, isArchived, isPinned, isMuted bool
	var createdAt, lastActivity, lastMessage, mutedUntil int64
	var groupInviteLink *string
	var messageCount, unreadCount int64

	err := row.Scan(&name, &description, &isGroup, &isBroadcast, &createdAt, &lastActivity, &lastMessage,
		&isArchived, &isPinned, &isMuted, &mutedUntil, &groupInviteLink, &messageCount, &unreadCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan conversation: %w", err)
	}

	conv := &types.ConversationInfo{
		ChatJID:         chat,
		Name:            name,
		Description:     description,
		IsGroup:         isGroup,
		IsBroadcast:     isBroadcast,
		CreatedAt:       time.Unix(createdAt, 0),
		LastActivity:    time.Unix(lastActivity, 0),
		IsArchived:      isArchived,
		IsPinned:        isPinned,
		IsMuted:         isMuted,
		GroupInviteLink: groupInviteLink,
		MessageCount:    messageCount,
		UnreadCount:     unreadCount,
	}

	if lastMessage > 0 {
		ts := time.Unix(lastMessage, 0)
		conv.LastMessage = &ts
	}

	if mutedUntil > 0 {
		ts := time.Unix(mutedUntil, 0)
		conv.MutedUntil = &ts
	}

	// Load group participants if this is a group
	if isGroup {
		participants, admins, err := s.getGroupParticipants(ctx, chat)
		if err != nil {
			return nil, fmt.Errorf("failed to get group participants: %w", err)
		}
		conv.GroupParticipants = participants
		conv.GroupAdmins = admins
	}

	return conv, nil
}

// GetAllConversations retrieves all conversations
func (s *ConversationStore) GetAllConversations(ctx context.Context) ([]*types.ConversationInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT chat_jid, name, description, is_group, is_broadcast, created_at, last_activity, last_message,
		       is_archived, is_pinned, is_muted, muted_until, group_invite_link, message_count, unread_count
		FROM whatsmeow_conversations
		WHERE our_jid = $1
		ORDER BY last_activity DESC
	`, s.JID)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*types.ConversationInfo
	for rows.Next() {
		var chatJIDStr, name, description *string
		var isGroup, isBroadcast, isArchived, isPinned, isMuted bool
		var createdAt, lastActivity, lastMessage, mutedUntil int64
		var groupInviteLink *string
		var messageCount, unreadCount int64

		err := rows.Scan(&chatJIDStr, &name, &description, &isGroup, &isBroadcast, &createdAt, &lastActivity, &lastMessage,
			&isArchived, &isPinned, &isMuted, &mutedUntil, &groupInviteLink, &messageCount, &unreadCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}

		chatJID, err := types.ParseJID(*chatJIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse chat JID: %w", err)
		}

		conv := &types.ConversationInfo{
			ChatJID:         chatJID,
			Name:            name,
			Description:     description,
			IsGroup:         isGroup,
			IsBroadcast:     isBroadcast,
			CreatedAt:       time.Unix(createdAt, 0),
			LastActivity:    time.Unix(lastActivity, 0),
			IsArchived:      isArchived,
			IsPinned:        isPinned,
			IsMuted:         isMuted,
			GroupInviteLink: groupInviteLink,
			MessageCount:    messageCount,
			UnreadCount:     unreadCount,
		}

		if lastMessage > 0 {
			ts := time.Unix(lastMessage, 0)
			conv.LastMessage = &ts
		}

		if mutedUntil > 0 {
			ts := time.Unix(mutedUntil, 0)
			conv.MutedUntil = &ts
		}

		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// UpdateConversation updates conversation metadata
func (s *ConversationStore) UpdateConversation(ctx context.Context, conv *types.ConversationInfo) error {
	return s.StoreConversation(ctx, conv)
}

// DeleteConversation deletes a conversation
func (s *ConversationStore) DeleteConversation(ctx context.Context, chat types.JID) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM whatsmeow_conversations
		WHERE our_jid = $1 AND chat_jid = $2
	`, s.JID, chat.String())

	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}

// ArchiveConversation archives or unarchives a conversation
func (s *ConversationStore) ArchiveConversation(ctx context.Context, chat types.JID, archived bool) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE whatsmeow_conversations
		SET is_archived = $3
		WHERE our_jid = $1 AND chat_jid = $2
	`, s.JID, chat.String(), archived)

	if err != nil {
		return fmt.Errorf("failed to archive conversation: %w", err)
	}

	return nil
}

// PinConversation pins or unpins a conversation
func (s *ConversationStore) PinConversation(ctx context.Context, chat types.JID, pinned bool) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE whatsmeow_conversations
		SET is_pinned = $3
		WHERE our_jid = $1 AND chat_jid = $2
	`, s.JID, chat.String(), pinned)

	if err != nil {
		return fmt.Errorf("failed to pin conversation: %w", err)
	}

	return nil
}

// MuteConversation mutes or unmutes a conversation
func (s *ConversationStore) MuteConversation(ctx context.Context, chat types.JID, mutedUntil *time.Time) error {
	var mutedUntilUnix *int64
	if mutedUntil != nil {
		ts := mutedUntil.Unix()
		mutedUntilUnix = &ts
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE whatsmeow_conversations
		SET is_muted = $3, muted_until = $4
		WHERE our_jid = $1 AND chat_jid = $2
	`, s.JID, chat.String(), mutedUntil != nil, mutedUntilUnix)

	if err != nil {
		return fmt.Errorf("failed to mute conversation: %w", err)
	}

	return nil
}

// UpdateGroupParticipants updates group participants
func (s *ConversationStore) UpdateGroupParticipants(ctx context.Context, chat types.JID, participants []types.JID) error {
	return s.updateGroupParticipants(ctx, chat, participants, nil)
}

// UpdateGroupAdmins updates group admins
func (s *ConversationStore) UpdateGroupAdmins(ctx context.Context, chat types.JID, admins []types.JID) error {
	// First get all participants
	participants, _, err := s.getGroupParticipants(ctx, chat)
	if err != nil {
		return fmt.Errorf("failed to get existing participants: %w", err)
	}

	return s.updateGroupParticipants(ctx, chat, participants, admins)
}

// UpdateGroupInviteLink updates the group invite link
func (s *ConversationStore) UpdateGroupInviteLink(ctx context.Context, chat types.JID, link *string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE whatsmeow_conversations
		SET group_invite_link = $3
		WHERE our_jid = $1 AND chat_jid = $2
	`, s.JID, chat.String(), link)

	if err != nil {
		return fmt.Errorf("failed to update group invite link: %w", err)
	}

	return nil
}

// updateGroupParticipants updates group participants and admins
func (s *ConversationStore) updateGroupParticipants(ctx context.Context, chat types.JID, participants []types.JID, admins []types.JID) error {
	// Delete existing participants
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM whatsmeow_group_participants
		WHERE our_jid = $1 AND group_jid = $2
	`, s.JID, chat.String())
	if err != nil {
		return fmt.Errorf("failed to delete existing participants: %w", err)
	}

	// Insert new participants
	for _, participant := range participants {
		isAdmin := false
		for _, admin := range admins {
			if admin == participant {
				isAdmin = true
				break
			}
		}

		_, err := s.db.ExecContext(ctx, `
			INSERT INTO whatsmeow_group_participants (our_jid, group_jid, participant_jid, is_admin, join_timestamp)
			VALUES ($1, $2, $3, $4, $5)
		`, s.JID, chat.String(), participant.String(), isAdmin, time.Now().Unix())
		if err != nil {
			return fmt.Errorf("failed to insert participant: %w", err)
		}
	}

	return nil
}

// getGroupParticipants retrieves group participants and admins
func (s *ConversationStore) getGroupParticipants(ctx context.Context, chat types.JID) ([]types.JID, []types.JID, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT participant_jid, is_admin
		FROM whatsmeow_group_participants
		WHERE our_jid = $1 AND group_jid = $2
	`, s.JID, chat.String())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query group participants: %w", err)
	}
	defer rows.Close()

	var participants, admins []types.JID
	for rows.Next() {
		var participantJIDStr string
		var isAdmin bool

		err := rows.Scan(&participantJIDStr, &isAdmin)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan participant: %w", err)
		}

		participantJID, err := types.ParseJID(participantJIDStr)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse participant JID: %w", err)
		}

		participants = append(participants, participantJID)
		if isAdmin {
			admins = append(admins, participantJID)
		}
	}

	return participants, admins, nil
}
