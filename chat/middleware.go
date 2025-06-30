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

// MessageStorageMiddleware handles automatic storage of messages
type MessageStorageMiddleware struct {
	device *store.Device
}

// NewMessageStorageMiddleware creates a new message storage middleware
func NewMessageStorageMiddleware(device *store.Device) *MessageStorageMiddleware {
	return &MessageStorageMiddleware{
		device: device,
	}
}

// HandleMessage processes and stores a message event
func (m *MessageStorageMiddleware) HandleMessage(ctx context.Context, evt *events.Message) error {
	// Convert event message to stored message
	storedMsg, err := m.convertEventToStoredMessage(evt)
	if err != nil {
		return fmt.Errorf("failed to convert event to stored message: %w", err)
	}

	// Store the message
	err = m.device.ChatHistory.StoreMessage(ctx, storedMsg)
	if err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	// Update conversation metadata
	err = m.updateConversationMetadata(ctx, evt)
	if err != nil {
		return fmt.Errorf("failed to update conversation metadata: %w", err)
	}

	// Index the message for search
	err = m.device.MessageSearch.IndexMessage(ctx, storedMsg)
	if err != nil {
		return fmt.Errorf("failed to index message: %w", err)
	}

	return nil
}

// HandleMessageEdit handles message edit events
func (m *MessageStorageMiddleware) HandleMessageEdit(ctx context.Context, evt *events.Message) error {
	// Get the original message
	originalMsg, err := m.device.ChatHistory.GetMessage(ctx, evt.Info.Chat, evt.Info.ID)
	if err != nil {
		return fmt.Errorf("failed to get original message: %w", err)
	}

	if originalMsg == nil {
		// If original message not found, store as new message
		return m.HandleMessage(ctx, evt)
	}

	// Update the message with edit information
	updatedMsg, err := m.convertEventToStoredMessage(evt)
	if err != nil {
		return fmt.Errorf("failed to convert edit event to stored message: %w", err)
	}

	// Mark as edited
	updatedMsg.IsEdited = true
	now := time.Now()
	updatedMsg.EditTimestamp = &now

	// Update the message
	err = m.device.ChatHistory.UpdateMessage(ctx, updatedMsg)
	if err != nil {
		return fmt.Errorf("failed to update edited message: %w", err)
	}

	// Re-index for search
	err = m.device.MessageSearch.IndexMessage(ctx, updatedMsg)
	if err != nil {
		return fmt.Errorf("failed to re-index edited message: %w", err)
	}

	return nil
}

// HandleMessageRevoke handles message revoke events
func (m *MessageStorageMiddleware) HandleMessageRevoke(ctx context.Context, evt *events.Message) error {
	// Get the original message
	originalMsg, err := m.device.ChatHistory.GetMessage(ctx, evt.Info.Chat, evt.Info.ID)
	if err != nil {
		return fmt.Errorf("failed to get original message: %w", err)
	}

	if originalMsg == nil {
		// If original message not found, ignore the revoke
		return nil
	}

	// Mark as revoked
	originalMsg.IsRevoked = true

	// Update the message
	err = m.device.ChatHistory.UpdateMessage(ctx, originalMsg)
	if err != nil {
		return fmt.Errorf("failed to update revoked message: %w", err)
	}

	// Remove from search index
	err = m.device.MessageSearch.RemoveMessageFromIndex(ctx, evt.Info.Chat, evt.Info.ID)
	if err != nil {
		return fmt.Errorf("failed to remove revoked message from index: %w", err)
	}

	return nil
}

// HandleReaction handles message reaction events
func (m *MessageStorageMiddleware) HandleReaction(ctx context.Context, reaction *events.Receipt) error {
	// Convert receipt to reaction
	msgReaction := &types.MessageReaction{
		MessageID: reaction.MessageIDs[0], // Assuming single message reaction
		ChatJID:   reaction.Chat,
		SenderJID: reaction.Sender,
		Timestamp: reaction.Timestamp,
	}

	// Determine emoji based on receipt type
	switch reaction.Type {
	case types.ReceiptTypeRead:
		// This is a read receipt, not a reaction
		return nil
	case types.ReceiptTypeDelivered:
		// This is a delivery receipt, not a reaction
		return nil
	default:
		// For now, we'll use a default emoji
		// In a real implementation, you'd parse the actual reaction data
		msgReaction.Emoji = "👍"
	}

	// Store the reaction
	err := m.device.ChatHistory.StoreReaction(ctx, msgReaction)
	if err != nil {
		return fmt.Errorf("failed to store reaction: %w", err)
	}

	return nil
}

// convertEventToStoredMessage converts an events.Message to a types.StoredMessage
func (m *MessageStorageMiddleware) convertEventToStoredMessage(evt *events.Message) (*types.StoredMessage, error) {
	// Determine message type
	messageType := m.extractMessageType(evt)

	// Extract media information
	mediaType, mediaID, mediaSize, mediaMimeType := m.extractMediaInfo(evt)

	// Extract reply information
	replyToMessageID, replyToSenderJID := m.extractReplyInfo(evt)

	// Extract thread information
	threadMessageID, threadSenderJID := m.extractThreadInfo(evt)

	// Create stored message
	storedMsg := &types.StoredMessage{
		ID:               evt.Info.ID,
		ChatJID:          evt.Info.Chat,
		SenderJID:        evt.Info.Sender,
		Timestamp:        evt.Info.Timestamp,
		Message:          evt.Message,
		Type:             messageType,
		IsFromMe:         evt.Info.IsFromMe,
		IsGroup:          evt.Info.IsGroup,
		IsEdited:         evt.IsEdit,
		Status:           types.MessageStatusSent, // Default status
		MediaType:        mediaType,
		MediaID:          mediaID,
		MediaSize:        mediaSize,
		MediaMimeType:    mediaMimeType,
		PushName:         &evt.Info.PushName,
		Category:         &evt.Info.Category,
		Multicast:        evt.Info.Multicast,
		ReplyToMessageID: replyToMessageID,
		ReplyToSenderJID: replyToSenderJID,
		ThreadMessageID:  threadMessageID,
		ThreadSenderJID:  threadSenderJID,
	}

	// Set verified name if available
	if evt.Info.VerifiedName != nil {
		storedMsg.VerifiedName = evt.Info.VerifiedName
	}

	return storedMsg, nil
}

// extractMessageType extracts the message type from the event
func (m *MessageStorageMiddleware) extractMessageType(evt *events.Message) string {
	// This is a simplified implementation
	// In a real implementation, you'd analyze the protobuf message structure
	if evt.Message.GetConversation() != "" {
		return "text"
	}
	if evt.Message.GetImageMessage() != nil {
		return "image"
	}
	if evt.Message.GetVideoMessage() != nil {
		return "video"
	}
	if evt.Message.GetAudioMessage() != nil {
		return "audio"
	}
	if evt.Message.GetDocumentMessage() != nil {
		return "document"
	}
	if evt.Message.GetStickerMessage() != nil {
		return "sticker"
	}
	if evt.Message.GetContactMessage() != nil {
		return "contact"
	}
	if evt.Message.GetLocationMessage() != nil {
		return "location"
	}
	if evt.Message.GetLiveLocationMessage() != nil {
		return "live_location"
	}
	if evt.Message.GetProtocolMessage() != nil {
		return "protocol"
	}
	if evt.Message.GetSenderKeyDistributionMessage() != nil {
		return "sender_key_distribution"
	}
	if evt.Message.GetDeviceSentMessage() != nil {
		return "device_sent"
	}
	if evt.Message.GetReactionMessage() != nil {
		return "reaction"
	}
	if evt.Message.GetEditedMessage() != nil {
		return "edited"
	}
	if evt.Message.GetEphemeralMessage() != nil {
		return "ephemeral"
	}
	if evt.Message.GetViewOnceMessage() != nil {
		return "view_once"
	}
	if evt.Message.GetViewOnceMessageV2() != nil {
		return "view_once_v2"
	}
	if evt.Message.GetViewOnceMessageV2Extension() != nil {
		return "view_once_v2_extension"
	}
	if evt.Message.GetDocumentWithCaptionMessage() != nil {
		return "document_with_caption"
	}
	if evt.Message.GetLottieStickerMessage() != nil {
		return "lottie_sticker"
	}
	if evt.Message.GetInteractiveMessage() != nil {
		return "interactive"
	}
	if evt.Message.GetContactsArrayMessage() != nil {
		return "contacts_array"
	}
	if evt.Message.GetHighlyStructuredMessage() != nil {
		return "highly_structured"
	}
	if evt.Message.GetFastRatchetKeySenderKeyDistributionMessage() != nil {
		return "fast_ratchet_key_sender_key_distribution"
	}
	if evt.Message.GetSendPaymentMessage() != nil {
		return "send_payment"
	}
	if evt.Message.GetRequestPaymentMessage() != nil {
		return "request_payment"
	}
	if evt.Message.GetDeclinePaymentRequestMessage() != nil {
		return "decline_payment_request"
	}
	if evt.Message.GetCancelPaymentRequestMessage() != nil {
		return "cancel_payment_request"
	}
	if evt.Message.GetTemplateMessage() != nil {
		return "template"
	}
	if evt.Message.GetStickerSyncRmrMessage() != nil {
		return "sticker_sync_rmr"
	}
	if evt.Message.GetInteractiveResponseMessage() != nil {
		return "interactive_response"
	}
	if evt.Message.GetPlaceholderMessage() != nil {
		return "placeholder"
	}

	return "unknown"
}

// extractMediaInfo extracts media information from the event
func (m *MessageStorageMiddleware) extractMediaInfo(evt *events.Message) (*string, *string, *int64, *string) {
	// This is a simplified implementation
	// In a real implementation, you'd extract actual media information from the protobuf message

	if evt.Message.GetImageMessage() != nil {
		imgMsg := evt.Message.GetImageMessage()
		mediaType := "image"
		mediaID := imgMsg.GetURL()
		mediaSize := int64(imgMsg.GetFileLength())
		mediaMimeType := imgMsg.GetMimetype()
		return &mediaType, &mediaID, &mediaSize, &mediaMimeType
	}

	if evt.Message.GetVideoMessage() != nil {
		vidMsg := evt.Message.GetVideoMessage()
		mediaType := "video"
		mediaID := vidMsg.GetURL()
		mediaSize := int64(vidMsg.GetFileLength())
		mediaMimeType := vidMsg.GetMimetype()
		return &mediaType, &mediaID, &mediaSize, &mediaMimeType
	}

	if evt.Message.GetAudioMessage() != nil {
		audMsg := evt.Message.GetAudioMessage()
		mediaType := "audio"
		mediaID := audMsg.GetURL()
		mediaSize := int64(audMsg.GetFileLength())
		mediaMimeType := audMsg.GetMimetype()
		return &mediaType, &mediaID, &mediaSize, &mediaMimeType
	}

	if evt.Message.GetDocumentMessage() != nil {
		docMsg := evt.Message.GetDocumentMessage()
		mediaType := "document"
		mediaID := docMsg.GetURL()
		mediaSize := int64(docMsg.GetFileLength())
		mediaMimeType := docMsg.GetMimetype()
		return &mediaType, &mediaID, &mediaSize, &mediaMimeType
	}

	if evt.Message.GetStickerMessage() != nil {
		stickerMsg := evt.Message.GetStickerMessage()
		mediaType := "sticker"
		mediaID := stickerMsg.GetURL()
		mediaSize := int64(stickerMsg.GetFileLength())
		mediaMimeType := stickerMsg.GetMimetype()
		return &mediaType, &mediaID, &mediaSize, &mediaMimeType
	}

	return nil, nil, nil, nil
}

// extractReplyInfo extracts reply information from the event
func (m *MessageStorageMiddleware) extractReplyInfo(evt *events.Message) (*types.MessageID, *types.JID) {
	// This is a simplified implementation
	// In a real implementation, you'd extract actual reply information from the protobuf message

	// Check for context info with quoted message
	if evt.Message.GetMessageContextInfo() != nil {
		// Note: The actual method name may vary depending on the protobuf definition
		// This is a placeholder for the actual implementation
		// contextInfo := evt.Message.GetMessageContextInfo()
		// quotedMsg := contextInfo.GetQuotedMessage()
		// if quotedMsg != nil && quotedMsg.GetKey() != nil {
		//     key := quotedMsg.GetKey()
		//     messageID := types.MessageID(key.GetID())
		//     senderJID, _ := types.ParseJID(key.GetParticipant())
		//     return &messageID, &senderJID
		// }
	}

	return nil, nil
}

// extractThreadInfo extracts thread information from the event
func (m *MessageStorageMiddleware) extractThreadInfo(evt *events.Message) (*types.MessageID, *types.JID) {
	// This is a simplified implementation
	// In a real implementation, you'd extract actual thread information from the protobuf message

	// Check for thread message ID in meta info
	if evt.Info.MsgMetaInfo.ThreadMessageID != "" {
		threadMessageID := evt.Info.MsgMetaInfo.ThreadMessageID
		threadSenderJID := evt.Info.MsgMetaInfo.ThreadMessageSenderJID
		return &threadMessageID, &threadSenderJID
	}

	return nil, nil
}

// updateConversationMetadata updates conversation metadata when a message is received
func (m *MessageStorageMiddleware) updateConversationMetadata(ctx context.Context, evt *events.Message) error {
	// Get existing conversation or create new one
	conv, err := m.device.Conversations.GetConversation(ctx, evt.Info.Chat)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	if conv == nil {
		// Create new conversation
		conv = &types.ConversationInfo{
			ChatJID:      evt.Info.Chat,
			IsGroup:      evt.Info.IsGroup,
			IsBroadcast:  evt.Info.Chat.Server == types.BroadcastServer,
			CreatedAt:    evt.Info.Timestamp,
			LastActivity: evt.Info.Timestamp,
			LastMessage:  &evt.Info.Timestamp,
			MessageCount: 1,
		}

		// Set name for individual chats
		if !evt.Info.IsGroup && !evt.Info.IsFromMe {
			conv.Name = &evt.Info.PushName
		}
	} else {
		// Update existing conversation
		conv.LastActivity = evt.Info.Timestamp
		conv.LastMessage = &evt.Info.Timestamp
		conv.MessageCount++

		// Update name if not set and this is an individual chat
		if !evt.Info.IsGroup && !evt.Info.IsFromMe && conv.Name == nil {
			conv.Name = &evt.Info.PushName
		}
	}

	// Store the conversation
	err = m.device.Conversations.StoreConversation(ctx, conv)
	if err != nil {
		return fmt.Errorf("failed to store conversation: %w", err)
	}

	return nil
}
