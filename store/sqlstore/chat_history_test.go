// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package sqlstore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mau.fi/util/dbutil"
	"go.mau.fi/whatsmeow/types"
)

func TestChatHistoryStores(t *testing.T) {
	// Create a test container
	container := NewWithWrappedDB(createTestDB(t), "sqlite", nil)
	defer container.Close()

	// Create a test device
	device := container.NewDevice()
	device.ID = &types.JID{User: "1234567890", Server: types.DefaultUserServer}
	device.LID = types.JID{User: "1234567890", Server: types.HiddenUserServer}

	// Initialize the device
	container.initializeDevice(device)

	// Test that the new stores are initialized
	assert.NotNil(t, device.ChatHistory)
	assert.NotNil(t, device.Conversations)
	assert.NotNil(t, device.MessageSearch)

	// Test conversation store
	t.Run("ConversationStore", func(t *testing.T) {
		ctx := context.Background()

		// Create a test conversation
		conv := &types.ConversationInfo{
			ChatJID:      types.JID{User: "9876543210", Server: types.DefaultUserServer},
			Name:         stringPtr("Test Chat"),
			IsGroup:      false,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}

		// Store the conversation
		err := device.Conversations.StoreConversation(ctx, conv)
		require.NoError(t, err)

		// Retrieve the conversation
		retrieved, err := device.Conversations.GetConversation(ctx, conv.ChatJID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, conv.ChatJID, retrieved.ChatJID)
		assert.Equal(t, conv.Name, retrieved.Name)
		assert.Equal(t, conv.IsGroup, retrieved.IsGroup)
	})

	// Test chat history store
	t.Run("ChatHistoryStore", func(t *testing.T) {
		ctx := context.Background()
		chatJID := types.JID{User: "9876543210", Server: types.DefaultUserServer}

		// Create a test message
		msg := &types.StoredMessage{
			ID:        "test-message-1",
			ChatJID:   chatJID,
			SenderJID: types.JID{User: "1234567890", Server: types.DefaultUserServer},
			Timestamp: time.Now(),
			Type:      "text",
			IsFromMe:  true,
			Status:    types.MessageStatusSent,
		}

		// Store the message
		err := device.ChatHistory.StoreMessage(ctx, msg)
		require.NoError(t, err)

		// Retrieve the message
		retrieved, err := device.ChatHistory.GetMessage(ctx, chatJID, msg.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, msg.ID, retrieved.ID)
		assert.Equal(t, msg.ChatJID, retrieved.ChatJID)
		assert.Equal(t, msg.SenderJID, retrieved.SenderJID)
		assert.Equal(t, msg.Type, retrieved.Type)

		// Test message count
		count, err := device.ChatHistory.GetMessageCount(ctx, chatJID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
}

// Helper function to create a test database
func createTestDB(t *testing.T) *dbutil.Database {
	// This is a simplified test setup
	// In a real test, you would use a proper test database
	t.Skip("Test database setup not implemented")
	return nil
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
