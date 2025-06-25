package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"go.mau.fi/whatsmeow/types"
)

// NeonizeStore contains methods for neonize-specific storage operations
type NeonizeStore struct {
	*NeonizeSQLStore
}

// Message represents a stored message
type Message struct {
	ID        string    `json:"id"`
	ChatJID   types.JID `json:"chat_jid"`
	SenderJID types.JID `json:"sender_jid"`
	Content   []byte    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// MessageReceipt represents a message receipt
type MessageReceipt struct {
	OurJID           types.JID
	ChatJID          types.JID
	MessageID        string
	RecipientJID     types.JID
	ReceiptType      string
	ReceiptTimestamp time.Time
}

// Media represents a stored media file
type Media struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Content   []byte    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// GroupMessage represents a stored group message
type GroupMessage struct {
	ID        string    `json:"id"`
	GroupJID  types.JID `json:"group_jid"`
	SenderJID types.JID `json:"sender_jid"`
	Content   []byte    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// GroupParticipant represents a group participant
type GroupParticipant struct {
	GroupJID  types.JID `json:"group_jid"`
	UserJID   types.JID `json:"user_jid"`
	IsAdmin   bool      `json:"is_admin"`
	Timestamp time.Time `json:"timestamp"`
}

// GroupInviteLink represents a group invite link
type GroupInviteLink struct {
	GroupJID  types.JID `json:"group_jid"`
	Link      string    `json:"link"`
	Timestamp time.Time `json:"timestamp"`
}

// NewsletterSubscription represents a newsletter subscription
type NewsletterSubscription struct {
	NewsletterJID types.JID `json:"newsletter_jid"`
	UserJID       types.JID `json:"user_jid"`
	Timestamp     time.Time `json:"timestamp"`
}

// NewsletterMessage represents a newsletter message
type NewsletterMessage struct {
	ID            string    `json:"id"`
	NewsletterJID types.JID `json:"newsletter_jid"`
	Content       []byte    `json:"content"`
	Timestamp     time.Time `json:"timestamp"`
}

// Call represents a stored call
type Call struct {
	ID        string    `json:"id"`
	CallerJID types.JID `json:"caller_jid"`
	CalleeJID types.JID `json:"callee_jid"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

// StoreMessage stores a message
func (s *NeonizeStore) StoreMessage(uuid []byte, messageBytes []byte) error {
	var msg Message
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO neonize_messages (uuid, id, chat_jid, sender_jid, content, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (uuid, id) DO UPDATE SET content=excluded.content
	`, uuid, msg.ID, msg.ChatJID.String(), msg.SenderJID.String(), msg.Content, msg.Timestamp)
	return err
}

// GetMessage retrieves a message
func (s *NeonizeStore) GetMessage(uuid []byte, chatJID types.JID, messageID string) (*Message, error) {
	var msg Message
	err := s.db.QueryRowContext(context.Background(), `
		SELECT id, chat_jid, sender_jid, content, timestamp
		FROM neonize_messages
		WHERE uuid = $1 AND id = $2
	`, uuid, messageID).Scan(&msg.ID, &msg.ChatJID, &msg.SenderJID, &msg.Content, &msg.Timestamp)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &msg, nil
}

// StoreMedia stores media
func (s *NeonizeStore) StoreMedia(uuid []byte, mediaBytes []byte) error {
	var media Media
	if err := json.Unmarshal(mediaBytes, &media); err != nil {
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO neonize_media (uuid, id, type, content, timestamp)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (uuid, id) DO UPDATE SET content=excluded.content
	`, uuid, media.ID, media.Type, media.Content, media.Timestamp)
	return err
}

// GetMedia retrieves media
func (s *NeonizeStore) GetMedia(uuid []byte, mediaID string) (*Media, error) {
	var media Media
	err := s.db.QueryRowContext(context.Background(), `
		SELECT id, type, content, timestamp
		FROM neonize_media
		WHERE uuid = $1 AND id = $2
	`, uuid, mediaID).Scan(&media.ID, &media.Type, &media.Content, &media.Timestamp)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &media, nil
}

// StoreGroupMessage stores a group message
func (s *NeonizeStore) StoreGroupMessage(uuid []byte, messageBytes []byte) error {
	var msg GroupMessage
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO neonize_group_messages (uuid, id, group_jid, sender_jid, content, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (uuid, id) DO UPDATE SET content=excluded.content
	`, uuid, msg.ID, msg.GroupJID.String(), msg.SenderJID.String(), msg.Content, msg.Timestamp)
	return err
}

// StoreGroupParticipant stores a group participant
func (s *NeonizeStore) StoreGroupParticipant(uuid []byte, participantBytes []byte) error {
	var participant GroupParticipant
	if err := json.Unmarshal(participantBytes, &participant); err != nil {
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO neonize_group_participants (uuid, group_jid, user_jid, is_admin, timestamp)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (uuid, group_jid, user_jid) DO UPDATE SET is_admin=excluded.is_admin
	`, uuid, participant.GroupJID.String(), participant.UserJID.String(), participant.IsAdmin, participant.Timestamp)
	return err
}

// StoreGroupInviteLink stores a group invite link
func (s *NeonizeStore) StoreGroupInviteLink(uuid []byte, linkBytes []byte) error {
	var link GroupInviteLink
	if err := json.Unmarshal(linkBytes, &link); err != nil {
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO neonize_group_invite_links (uuid, group_jid, link, timestamp)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (uuid, group_jid) DO UPDATE SET link=excluded.link
	`, uuid, link.GroupJID.String(), link.Link, link.Timestamp)
	return err
}

// StoreNewsletterSubscription stores a newsletter subscription
func (s *NeonizeStore) StoreNewsletterSubscription(uuid []byte, subBytes []byte) error {
	var sub NewsletterSubscription
	if err := json.Unmarshal(subBytes, &sub); err != nil {
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO neonize_newsletter_subscriptions (uuid, newsletter_jid, user_jid, timestamp)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (uuid, newsletter_jid, user_jid) DO NOTHING
	`, uuid, sub.NewsletterJID.String(), sub.UserJID.String(), sub.Timestamp)
	return err
}

// StoreNewsletterMessage stores a newsletter message
func (s *NeonizeStore) StoreNewsletterMessage(uuid []byte, messageBytes []byte) error {
	var msg NewsletterMessage
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO neonize_newsletter_messages (uuid, id, newsletter_jid, content, timestamp)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (uuid, id) DO UPDATE SET content=excluded.content
	`, uuid, msg.ID, msg.NewsletterJID.String(), msg.Content, msg.Timestamp)
	return err
}

// StoreCall stores a call
func (s *NeonizeStore) StoreCall(uuid []byte, callBytes []byte) error {
	var call Call
	if err := json.Unmarshal(callBytes, &call); err != nil {
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `
		INSERT INTO neonize_calls (uuid, id, caller_jid, callee_jid, type, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (uuid, id) DO UPDATE SET type=excluded.type
	`, uuid, call.ID, call.CallerJID.String(), call.CalleeJID.String(), call.Type, call.Timestamp)
	return err
}

// PutMessageReceipt stores a message receipt in the database
func (s *NeonizeStore) PutMessageReceipt(receipt *MessageReceipt) error {
	_, err := s.db.Exec(`
		INSERT INTO whatsmeow_message_receipts (
			our_jid, chat_jid, message_id, recipient_jid,
			receipt_type, receipt_timestamp
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (our_jid, chat_jid, message_id, recipient_jid) DO UPDATE SET
			receipt_type = $5,
			receipt_timestamp = $6
	`,
		receipt.OurJID.String(), receipt.ChatJID.String(),
		receipt.MessageID, receipt.RecipientJID.String(),
		receipt.ReceiptType, receipt.ReceiptTimestamp,
	)
	return err
}
