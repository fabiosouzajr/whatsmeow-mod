// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package types

import (
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

// StoredMessage represents a message stored in the chat history
type StoredMessage struct {
	ID        MessageID
	ChatJID   JID
	SenderJID JID
	Timestamp time.Time

	// Message content
	Message *waE2E.Message
	Type    string

	// Message metadata
	IsFromMe      bool
	IsGroup       bool
	IsEdited      bool
	IsRevoked     bool
	EditTimestamp *time.Time

	// Reply information
	ReplyToMessageID *MessageID
	ReplyToSenderJID *JID

	// Thread information
	ThreadMessageID *MessageID
	ThreadSenderJID *JID

	// Message status
	Status MessageStatus

	// Media information (if applicable)
	MediaType     *string
	MediaID       *string
	MediaSize     *int64
	MediaMimeType *string

	// Additional metadata
	PushName     *string
	VerifiedName *VerifiedName
	Category     *string
	Multicast    bool
}

// MessageStatus represents the status of a message
type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

// ConversationInfo represents metadata about a conversation
type ConversationInfo struct {
	ChatJID JID

	// Basic info
	Name        *string
	Description *string
	IsGroup     bool
	IsBroadcast bool

	// Timestamps
	CreatedAt    time.Time
	LastActivity time.Time
	LastMessage  *time.Time

	// Settings
	IsArchived bool
	IsPinned   bool
	IsMuted    bool
	MutedUntil *time.Time

	// Group-specific fields
	GroupParticipants []JID
	GroupAdmins       []JID
	GroupInviteLink   *string

	// Message counts
	MessageCount int64
	UnreadCount  int64
}

// MessageSearchResult represents a search result
type MessageSearchResult struct {
	Message    *StoredMessage
	Relevance  float64
	Highlights []string // Text snippets where search terms were found
}

// ChatHistoryQuery represents a query for retrieving chat history
type ChatHistoryQuery struct {
	ChatJID     JID
	Limit       int
	Before      *time.Time
	After       *time.Time
	FromSender  *JID
	MessageType *string
	SearchTerm  *string
}

// MessageReaction represents a reaction to a message
type MessageReaction struct {
	MessageID MessageID
	ChatJID   JID
	SenderJID JID
	Emoji     string
	Timestamp time.Time
}

// MessageForward represents forwarding metadata
type MessageForward struct {
	OriginalMessageID MessageID
	OriginalChatJID   JID
	OriginalSenderJID JID
	OriginalTimestamp time.Time
	ForwardTimestamp  time.Time
}
