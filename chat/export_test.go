// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chat

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

// MockDevice implements a mock device for testing
type MockDevice struct {
	conversations map[types.JID]*types.ConversationInfo
	messages      map[types.JID][]*types.StoredMessage
	reactions     map[string][]*types.MessageReaction
	forwards      map[string]*types.MessageForward
}

func NewMockDevice() *MockDevice {
	return &MockDevice{
		conversations: make(map[types.JID]*types.ConversationInfo),
		messages:      make(map[types.JID][]*types.StoredMessage),
		reactions:     make(map[string][]*types.MessageReaction),
		forwards:      make(map[string]*types.MessageForward),
	}
}

func (m *MockDevice) Conversations() store.ConversationStore {
	return &MockConversationStore{device: m}
}

func (m *MockDevice) ChatHistory() store.ChatHistoryStore {
	return &MockChatHistoryStore{device: m}
}

// MockConversationStore implements store.ConversationStore for testing
type MockConversationStore struct {
	device *MockDevice
}

func (m *MockConversationStore) GetConversation(ctx context.Context, chatJID types.JID) (*types.ConversationInfo, error) {
	if conv, exists := m.device.conversations[chatJID]; exists {
		return conv, nil
	}
	return nil, nil
}

func (m *MockConversationStore) StoreConversation(ctx context.Context, conv *types.ConversationInfo) error {
	m.device.conversations[conv.ChatJID] = conv
	return nil
}

func (m *MockConversationStore) GetAllConversations(ctx context.Context) ([]*types.ConversationInfo, error) {
	var conversations []*types.ConversationInfo
	for _, conv := range m.device.conversations {
		conversations = append(conversations, conv)
	}
	return conversations, nil
}

func (m *MockConversationStore) DeleteConversation(ctx context.Context, chatJID types.JID) error {
	delete(m.device.conversations, chatJID)
	return nil
}

func (m *MockConversationStore) UpdateConversation(ctx context.Context, conv *types.ConversationInfo) error {
	m.device.conversations[conv.ChatJID] = conv
	return nil
}

func (m *MockConversationStore) ArchiveConversation(ctx context.Context, chat types.JID, archived bool) error {
	if conv, exists := m.device.conversations[chat]; exists {
		conv.IsArchived = archived
		m.device.conversations[chat] = conv
	}
	return nil
}

func (m *MockConversationStore) PinConversation(ctx context.Context, chat types.JID, pinned bool) error {
	if conv, exists := m.device.conversations[chat]; exists {
		conv.IsPinned = pinned
		m.device.conversations[chat] = conv
	}
	return nil
}

func (m *MockConversationStore) MuteConversation(ctx context.Context, chat types.JID, mutedUntil *time.Time) error {
	if conv, exists := m.device.conversations[chat]; exists {
		conv.IsMuted = mutedUntil != nil
		conv.MutedUntil = mutedUntil
		m.device.conversations[chat] = conv
	}
	return nil
}

func (m *MockConversationStore) UpdateGroupParticipants(ctx context.Context, chat types.JID, participants []types.JID) error {
	if conv, exists := m.device.conversations[chat]; exists {
		conv.GroupParticipants = participants
		m.device.conversations[chat] = conv
	}
	return nil
}

func (m *MockConversationStore) UpdateGroupAdmins(ctx context.Context, chat types.JID, admins []types.JID) error {
	if conv, exists := m.device.conversations[chat]; exists {
		conv.GroupAdmins = admins
		m.device.conversations[chat] = conv
	}
	return nil
}

func (m *MockConversationStore) UpdateGroupInviteLink(ctx context.Context, chat types.JID, link *string) error {
	if conv, exists := m.device.conversations[chat]; exists {
		conv.GroupInviteLink = link
		m.device.conversations[chat] = conv
	}
	return nil
}

// MockChatHistoryStore implements store.ChatHistoryStore for testing
type MockChatHistoryStore struct {
	device *MockDevice
}

func (m *MockChatHistoryStore) StoreMessage(ctx context.Context, msg *types.StoredMessage) error {
	m.device.messages[msg.ChatJID] = append(m.device.messages[msg.ChatJID], msg)
	return nil
}

func (m *MockChatHistoryStore) GetMessages(ctx context.Context, query types.ChatHistoryQuery) ([]*types.StoredMessage, error) {
	messages, exists := m.device.messages[query.ChatJID]
	if !exists {
		return []*types.StoredMessage{}, nil
	}
	return messages, nil
}

func (m *MockChatHistoryStore) GetMessage(ctx context.Context, chat types.JID, messageID types.MessageID) (*types.StoredMessage, error) {
	messages, exists := m.device.messages[chat]
	if !exists {
		return nil, nil
	}
	for _, msg := range messages {
		if msg.ID == messageID {
			return msg, nil
		}
	}
	return nil, nil
}

func (m *MockChatHistoryStore) UpdateMessage(ctx context.Context, msg *types.StoredMessage) error {
	return m.StoreMessage(ctx, msg)
}

func (m *MockChatHistoryStore) DeleteMessage(ctx context.Context, chat types.JID, messageID types.MessageID) error {
	messages, exists := m.device.messages[chat]
	if !exists {
		return nil
	}
	for i, msg := range messages {
		if msg.ID == messageID {
			m.device.messages[chat] = append(messages[:i], messages[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockChatHistoryStore) GetMessageCount(ctx context.Context, chat types.JID) (int64, error) {
	messages, exists := m.device.messages[chat]
	if !exists {
		return 0, nil
	}
	return int64(len(messages)), nil
}

func (m *MockChatHistoryStore) GetLastMessage(ctx context.Context, chat types.JID) (*types.StoredMessage, error) {
	messages, exists := m.device.messages[chat]
	if !exists || len(messages) == 0 {
		return nil, nil
	}
	return messages[len(messages)-1], nil
}

func (m *MockChatHistoryStore) StoreReaction(ctx context.Context, reaction *types.MessageReaction) error {
	key := string(reaction.MessageID)
	m.device.reactions[key] = append(m.device.reactions[key], reaction)
	return nil
}

func (m *MockChatHistoryStore) GetReactions(ctx context.Context, chat types.JID, messageID types.MessageID) ([]*types.MessageReaction, error) {
	key := string(messageID)
	reactions, exists := m.device.reactions[key]
	if !exists {
		return []*types.MessageReaction{}, nil
	}
	return reactions, nil
}

func (m *MockChatHistoryStore) DeleteReaction(ctx context.Context, chat types.JID, messageID types.MessageID, sender types.JID) error {
	key := string(messageID)
	reactions, exists := m.device.reactions[key]
	if !exists {
		return nil
	}
	for i, reaction := range reactions {
		if reaction.SenderJID == sender {
			m.device.reactions[key] = append(reactions[:i], reactions[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockChatHistoryStore) StoreForward(ctx context.Context, forward *types.MessageForward) error {
	key := string(forward.OriginalMessageID)
	m.device.forwards[key] = forward
	return nil
}

func (m *MockChatHistoryStore) GetForward(ctx context.Context, chat types.JID, messageID types.MessageID) (*types.MessageForward, error) {
	key := string(messageID)
	forward, exists := m.device.forwards[key]
	if !exists {
		return nil, nil
	}
	return forward, nil
}

// Test helper functions
func createTestMessage(id string, chatJID types.JID, senderJID types.JID, msgType string, content string, timestamp time.Time) *types.StoredMessage {
	return &types.StoredMessage{
		ID:        types.MessageID(id),
		ChatJID:   chatJID,
		SenderJID: senderJID,
		Timestamp: timestamp,
		Type:      msgType,
		Status:    types.MessageStatusDelivered,
		IsFromMe:  senderJID.Server == types.DefaultUserServer,
		IsGroup:   chatJID.Server == types.GroupServer,
	}
}

func createTestConversation(chatJID types.JID, name string) *types.ConversationInfo {
	convName := name
	return &types.ConversationInfo{
		ChatJID:      chatJID,
		Name:         &convName,
		IsGroup:      chatJID.Server == types.GroupServer,
		CreatedAt:    time.Now().AddDate(0, 0, -30),
		LastActivity: time.Now(),
		MessageCount: 10,
	}
}

// TestExportChatHistory tests the main export functionality
func TestExportChatHistory(t *testing.T) {
	// Create mock device
	mockDevice := NewMockDevice()
	exporter := MockMessageExporter(mockDevice)

	// Create test data
	chatJID, _ := types.ParseJID("1234567890@s.whatsapp.net")
	senderJID, _ := types.ParseJID("9876543210@s.whatsapp.net")

	// Add test conversation
	conv := createTestConversation(chatJID, "Test Chat")
	mockDevice.conversations[chatJID] = conv

	// Add test messages
	now := time.Now()
	messages := []*types.StoredMessage{
		createTestMessage("msg1", chatJID, senderJID, "text", "Hello world", now.Add(-2*time.Hour)),
		createTestMessage("msg2", chatJID, senderJID, "image", "", now.Add(-1*time.Hour)),
		createTestMessage("msg3", chatJID, senderJID, "text", "How are you?", now),
	}
	mockDevice.messages[chatJID] = messages

	// Add test reactions
	reaction := &types.MessageReaction{
		MessageID: "msg1",
		ChatJID:   chatJID,
		SenderJID: senderJID,
		Emoji:     "👍",
		Timestamp: now.Add(-1 * time.Hour),
	}
	mockDevice.reactions["msg1"] = []*types.MessageReaction{reaction}

	// Test JSON export
	t.Run("JSON Export", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "export.json")

		options := &ExportOptions{
			Format:           ExportFormatJSON,
			IncludeReactions: true,
			IncludeForwards:  false,
		}

		err := exporter.ExportChatHistory(context.Background(), chatJID, outputPath, options)
		if err != nil {
			t.Fatalf("Failed to export JSON: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatal("Export file was not created")
		}

		// Verify JSON content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read export file: %v", err)
		}

		var exportData ExportData
		err = json.Unmarshal(content, &exportData)
		if err != nil {
			t.Fatalf("Failed to parse JSON export: %v", err)
		}

		// Verify export data
		if exportData.Conversation == nil {
			t.Error("Conversation info not included in export")
		}
		if len(exportData.Messages) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(exportData.Messages))
		}
		if len(exportData.Reactions) != 1 {
			t.Errorf("Expected 1 reaction, got %d", len(exportData.Reactions))
		}
		if exportData.ExportInfo.TotalMessages != 3 {
			t.Errorf("Expected 3 total messages, got %d", exportData.ExportInfo.TotalMessages)
		}
	})

	// Test TXT export
	t.Run("TXT Export", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "export.txt")

		options := &ExportOptions{
			Format: ExportFormatTXT,
		}

		err := exporter.ExportChatHistory(context.Background(), chatJID, outputPath, options)
		if err != nil {
			t.Fatalf("Failed to export TXT: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatal("Export file was not created")
		}

		// Verify content contains expected text
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read export file: %v", err)
		}

		contentStr := string(content)
		if !contains(contentStr, "WhatsApp Chat History Export") {
			t.Error("TXT export missing header")
		}
		if !contains(contentStr, "Hello world") {
			t.Error("TXT export missing message content")
		}
		if !contains(contentStr, "Test Chat") {
			t.Error("TXT export missing conversation name")
		}
	})

	// Test CSV export
	t.Run("CSV Export", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "export.csv")

		options := &ExportOptions{
			Format: ExportFormatCSV,
		}

		err := exporter.ExportChatHistory(context.Background(), chatJID, outputPath, options)
		if err != nil {
			t.Fatalf("Failed to export CSV: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatal("Export file was not created")
		}

		// Verify content contains expected CSV structure
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read export file: %v", err)
		}

		contentStr := string(content)
		if !contains(contentStr, "Timestamp,Sender,Type,Content") {
			t.Error("CSV export missing header")
		}
		if !contains(contentStr, "text") {
			t.Error("CSV export missing message type")
		}
		if !contains(contentStr, "image") {
			t.Error("CSV export missing image message type")
		}
	})
}

// TestExportWithFilters tests export with various filter options
func TestExportWithFilters(t *testing.T) {
	// Create mock device
	mockDevice := NewMockDevice()
	exporter := MockMessageExporter(mockDevice)

	chatJID, _ := types.ParseJID("1234567890@s.whatsapp.net")
	senderJID, _ := types.ParseJID("9876543210@s.whatsapp.net")

	// Add test conversation
	conv := createTestConversation(chatJID, "Test Chat")
	mockDevice.conversations[chatJID] = conv

	// Add test messages with different timestamps
	now := time.Now()
	messages := []*types.StoredMessage{
		createTestMessage("msg1", chatJID, senderJID, "text", "Old message", now.AddDate(0, 0, -10)),
		createTestMessage("msg2", chatJID, senderJID, "text", "Recent message", now.AddDate(0, 0, -1)),
		createTestMessage("msg3", chatJID, senderJID, "image", "", now),
	}
	mockDevice.messages[chatJID] = messages

	t.Run("Date Range Filter", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "filtered_export.json")

		options := &ExportOptions{
			Format: ExportFormatJSON,
			DateRange: &DateRange{
				Start: now.AddDate(0, 0, -5), // Only last 5 days
				End:   now,
			},
		}

		err := exporter.ExportChatHistory(context.Background(), chatJID, outputPath, options)
		if err != nil {
			t.Fatalf("Failed to export with date filter: %v", err)
		}

		// Verify filtered content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read export file: %v", err)
		}

		var exportData ExportData
		err = json.Unmarshal(content, &exportData)
		if err != nil {
			t.Fatalf("Failed to parse JSON export: %v", err)
		}

		// Should only include recent messages (last 5 days)
		if len(exportData.Messages) != 2 {
			t.Errorf("Expected 2 messages with date filter, got %d", len(exportData.Messages))
		}
	})

	t.Run("Message Type Filter", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "text_only_export.json")

		msgType := "text"
		options := &ExportOptions{
			Format:      ExportFormatJSON,
			MessageType: &msgType,
		}

		err := exporter.ExportChatHistory(context.Background(), chatJID, outputPath, options)
		if err != nil {
			t.Fatalf("Failed to export with type filter: %v", err)
		}

		// Verify filtered content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read export file: %v", err)
		}

		var exportData ExportData
		err = json.Unmarshal(content, &exportData)
		if err != nil {
			t.Fatalf("Failed to parse JSON export: %v", err)
		}

		// Should only include text messages
		for _, msg := range exportData.Messages {
			if msg.Type != "text" {
				t.Errorf("Expected only text messages, got %s", msg.Type)
			}
		}
	})
}

// TestExportMedia tests media export functionality
func TestExportMedia(t *testing.T) {
	// Create mock device
	mockDevice := NewMockDevice()
	exporter := MockMessageExporter(mockDevice)

	chatJID, _ := types.ParseJID("1234567890@s.whatsapp.net")
	senderJID, _ := types.ParseJID("9876543210@s.whatsapp.net")

	// Add test messages with media
	now := time.Now()
	mediaID := "media123"
	mediaType := "image"
	messages := []*types.StoredMessage{
		{
			ID:        "msg1",
			ChatJID:   chatJID,
			SenderJID: senderJID,
			Timestamp: now,
			Type:      "image",
			MediaID:   &mediaID,
			MediaType: &mediaType,
			Status:    types.MessageStatusDelivered,
		},
	}
	mockDevice.messages[chatJID] = messages

	t.Run("Media Export", func(t *testing.T) {
		tempDir := t.TempDir()
		outputDir := filepath.Join(tempDir, "media_export")

		options := &ExportOptions{
			IncludeMedia: true,
		}

		err := exporter.ExportMedia(context.Background(), chatJID, outputDir, options)
		if err != nil {
			t.Fatalf("Failed to export media: %v", err)
		}

		// Verify output directory was created
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			t.Fatal("Media export directory was not created")
		}
	})
}

// TestExportErrorHandling tests error scenarios
func TestExportErrorHandling(t *testing.T) {
	// Create mock device
	mockDevice := NewMockDevice()
	exporter := MockMessageExporter(mockDevice)

	chatJID, _ := types.ParseJID("1234567890@s.whatsapp.net")

	t.Run("Non-existent Chat", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "export.json")

		options := &ExportOptions{
			Format: ExportFormatJSON,
		}

		err := exporter.ExportChatHistory(context.Background(), chatJID, outputPath, options)
		if err == nil {
			t.Error("Expected error for non-existent chat, got nil")
		}
	})

	t.Run("Invalid Format", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "export.xyz")

		options := &ExportOptions{
			Format: "invalid",
		}

		// Add test data
		conv := createTestConversation(chatJID, "Test Chat")
		mockDevice.conversations[chatJID] = conv
		mockDevice.messages[chatJID] = []*types.StoredMessage{
			createTestMessage("msg1", chatJID, types.EmptyJID, "text", "Hello", time.Now()),
		}

		err := exporter.ExportChatHistory(context.Background(), chatJID, outputPath, options)
		if err == nil {
			t.Error("Expected error for invalid format, got nil")
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}

// MockMessageExporter creates a mock message exporter for testing
func MockMessageExporter(device *MockDevice) *MessageExporter {
	// Create a wrapper that satisfies the store.Device interface
	deviceWrapper := &store.Device{
		ChatHistory:   device.ChatHistory(),
		Conversations: device.Conversations(),
	}
	return &MessageExporter{
		device: deviceWrapper,
	}
}
