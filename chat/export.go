// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chat

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

// ExportFormat represents the supported export formats
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatTXT  ExportFormat = "txt"
	ExportFormatCSV  ExportFormat = "csv"
)

// ExportOptions contains configuration for message export
type ExportOptions struct {
	Format           ExportFormat
	IncludeMedia     bool
	DateRange        *DateRange
	FromSender       *types.JID
	MessageType      *string
	IncludeReactions bool
	IncludeForwards  bool
}

// DateRange represents a date range for filtering
type DateRange struct {
	Start time.Time
	End   time.Time
}

// ExportData represents the structure of exported data
type ExportData struct {
	Conversation *types.ConversationInfo             `json:"conversation"`
	Messages     []*types.StoredMessage              `json:"messages"`
	Reactions    map[string][]*types.MessageReaction `json:"reactions,omitempty"`
	Forwards     map[string]*types.MessageForward    `json:"forwards,omitempty"`
	ExportInfo   ExportInfo                          `json:"export_info"`
}

// ExportInfo contains metadata about the export
type ExportInfo struct {
	ExportTime    time.Time  `json:"export_time"`
	TotalMessages int        `json:"total_messages"`
	DateRange     *DateRange `json:"date_range,omitempty"`
	Format        string     `json:"format"`
	Version       string     `json:"version"`
}

// MessageExporter handles message export functionality
type MessageExporter struct {
	device *store.Device
}

// NewMessageExporter creates a new message exporter
func NewMessageExporter(device *store.Device) *MessageExporter {
	return &MessageExporter{
		device: device,
	}
}

// ExportChatHistory exports chat history to the specified format
func (e *MessageExporter) ExportChatHistory(ctx context.Context, chatJID types.JID, outputPath string, options *ExportOptions) error {
	// Validate format
	if options == nil {
		options = &ExportOptions{Format: ExportFormatJSON}
	}

	// Get conversation info
	conv, err := e.device.Conversations.GetConversation(ctx, chatJID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	// Build query for messages
	query := types.ChatHistoryQuery{
		ChatJID: chatJID,
		Limit:   10000, // Large limit for export
	}

	if options.DateRange != nil {
		query.After = &options.DateRange.Start
		query.Before = &options.DateRange.End
	}

	if options.FromSender != nil {
		query.FromSender = options.FromSender
	}

	if options.MessageType != nil {
		query.MessageType = options.MessageType
	}

	// Get messages
	messages, err := e.device.ChatHistory.GetMessages(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	// Prepare export data
	exportData := &ExportData{
		Conversation: conv,
		Messages:     messages,
		ExportInfo: ExportInfo{
			ExportTime:    time.Now(),
			TotalMessages: len(messages),
			DateRange:     options.DateRange,
			Format:        string(options.Format),
			Version:       "1.0",
		},
	}

	// Add reactions if requested
	if options.IncludeReactions {
		exportData.Reactions = make(map[string][]*types.MessageReaction)
		for _, msg := range messages {
			reactions, err := e.device.ChatHistory.GetReactions(ctx, chatJID, msg.ID)
			if err == nil && len(reactions) > 0 {
				exportData.Reactions[string(msg.ID)] = reactions
			}
		}
	}

	// Add forwards if requested
	if options.IncludeForwards {
		exportData.Forwards = make(map[string]*types.MessageForward)
		for _, msg := range messages {
			forward, err := e.device.ChatHistory.GetForward(ctx, chatJID, msg.ID)
			if err == nil && forward != nil {
				exportData.Forwards[string(msg.ID)] = forward
			}
		}
	}

	// Export based on format
	switch options.Format {
	case ExportFormatJSON:
		return e.exportToJSON(exportData, outputPath)
	case ExportFormatTXT:
		return e.exportToText(exportData, outputPath)
	case ExportFormatCSV:
		return e.exportToCSV(exportData, outputPath)
	default:
		return fmt.Errorf("unsupported export format: %s", options.Format)
	}
}

// exportToJSON exports chat history to JSON format
func (e *MessageExporter) exportToJSON(data *ExportData, outputPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON with pretty formatting
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

// exportToText exports chat history to text format
func (e *MessageExporter) exportToText(data *ExportData, outputPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create text file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "WhatsApp Chat History Export\n")
	fmt.Fprintf(file, "============================\n\n")

	if data.Conversation != nil {
		fmt.Fprintf(file, "Conversation: %s\n", data.Conversation.ChatJID)
		if data.Conversation.Name != nil {
			fmt.Fprintf(file, "Name: %s\n", *data.Conversation.Name)
		}
		fmt.Fprintf(file, "Type: %s\n", e.getConversationType(data.Conversation))
		fmt.Fprintf(file, "Created: %s\n", data.Conversation.CreatedAt.Format(time.RFC3339))
		fmt.Fprintf(file, "Last Activity: %s\n", data.Conversation.LastActivity.Format(time.RFC3339))
		fmt.Fprintf(file, "Message Count: %d\n", data.Conversation.MessageCount)
	}

	fmt.Fprintf(file, "\nExport Info:\n")
	fmt.Fprintf(file, "- Export Time: %s\n", data.ExportInfo.ExportTime.Format(time.RFC3339))
	fmt.Fprintf(file, "- Total Messages: %d\n", data.ExportInfo.TotalMessages)
	fmt.Fprintf(file, "- Format: %s\n", data.ExportInfo.Format)

	if data.ExportInfo.DateRange != nil {
		fmt.Fprintf(file, "- Date Range: %s to %s\n",
			data.ExportInfo.DateRange.Start.Format(time.RFC3339),
			data.ExportInfo.DateRange.End.Format(time.RFC3339))
	}

	fmt.Fprintf(file, "\n"+strings.Repeat("=", 80)+"\n\n")

	// Write messages
	for i, msg := range data.Messages {
		fmt.Fprintf(file, "[%d] %s - %s (%s)\n",
			i+1,
			msg.Timestamp.Format("2006-01-02 15:04:05"),
			msg.SenderJID,
			msg.Type)

		// Add message content
		if msg.Message != nil {
			content := e.extractMessageContent(msg)
			if content != "" {
				fmt.Fprintf(file, "Content: %s\n", content)
			}
		}

		// Add reactions if available
		if reactions, exists := data.Reactions[string(msg.ID)]; exists && len(reactions) > 0 {
			fmt.Fprintf(file, "Reactions: ")
			for j, reaction := range reactions {
				if j > 0 {
					fmt.Fprintf(file, ", ")
				}
				fmt.Fprintf(file, "%s %s", reaction.Emoji, reaction.SenderJID)
			}
			fmt.Fprintf(file, "\n")
		}

		// Add forward info if available
		if forward, exists := data.Forwards[string(msg.ID)]; exists {
			fmt.Fprintf(file, "Forwarded from: %s (original: %s)\n",
				forward.OriginalChatJID, forward.OriginalMessageID)
		}

		fmt.Fprintf(file, "\n")
	}

	return nil
}

// exportToCSV exports chat history to CSV format
func (e *MessageExporter) exportToCSV(data *ExportData, outputPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Timestamp", "Sender", "Type", "Content", "IsFromMe", "IsGroup",
		"IsEdited", "IsRevoked", "Status", "MediaType", "MediaID",
		"ReplyToMessageID", "ThreadMessageID", "Reactions", "ForwardedFrom",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write messages
	for _, msg := range data.Messages {
		// Extract message content
		content := ""
		if msg.Message != nil {
			content = e.extractMessageContent(msg)
		}

		// Get reactions
		reactions := ""
		if msgReactions, exists := data.Reactions[string(msg.ID)]; exists {
			var reactionStrs []string
			for _, reaction := range msgReactions {
				reactionStrs = append(reactionStrs, fmt.Sprintf("%s:%s", reaction.Emoji, reaction.SenderJID))
			}
			reactions = strings.Join(reactionStrs, ";")
		}

		// Get forward info
		forwardedFrom := ""
		if forward, exists := data.Forwards[string(msg.ID)]; exists {
			forwardedFrom = forward.OriginalChatJID.String()
		}

		// Prepare row
		row := []string{
			msg.Timestamp.Format(time.RFC3339),
			msg.SenderJID.String(),
			msg.Type,
			content,
			fmt.Sprintf("%t", msg.IsFromMe),
			fmt.Sprintf("%t", msg.IsGroup),
			fmt.Sprintf("%t", msg.IsEdited),
			fmt.Sprintf("%t", msg.IsRevoked),
			string(msg.Status),
			e.getStringValue(msg.MediaType),
			e.getStringValue(msg.MediaID),
			e.getMessageIDValue(msg.ReplyToMessageID),
			e.getMessageIDValue(msg.ThreadMessageID),
			reactions,
			forwardedFrom,
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// extractMessageContent extracts readable content from a message
func (e *MessageExporter) extractMessageContent(msg *types.StoredMessage) string {
	if msg.Message == nil {
		return ""
	}

	// This is a simplified implementation
	// In a real implementation, you'd extract text from the protobuf message based on type

	switch msg.Type {
	case "text":
		// Extract conversation text
		if msg.Message.GetConversation() != "" {
			return msg.Message.GetConversation()
		}
	case "image":
		return "[Image]"
	case "video":
		return "[Video]"
	case "audio":
		return "[Audio]"
	case "document":
		return "[Document]"
	case "sticker":
		return "[Sticker]"
	case "contact":
		return "[Contact]"
	case "location":
		return "[Location]"
	default:
		return fmt.Sprintf("[%s]", msg.Type)
	}

	return ""
}

// getConversationType returns a human-readable conversation type
func (e *MessageExporter) getConversationType(conv *types.ConversationInfo) string {
	if conv == nil {
		return "Unknown"
	}

	if conv.IsGroup {
		return "Group"
	}
	if conv.IsBroadcast {
		return "Broadcast"
	}
	return "Individual"
}

// getStringValue safely returns a string value from a pointer
func (e *MessageExporter) getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// getMessageIDValue safely returns a string value from a MessageID pointer
func (e *MessageExporter) getMessageIDValue(id *types.MessageID) string {
	if id == nil {
		return ""
	}
	return string(*id)
}

// ExportMedia exports media files from messages
func (e *MessageExporter) ExportMedia(ctx context.Context, chatJID types.JID, outputDir string, options *ExportOptions) error {
	// Build query for messages with media
	query := types.ChatHistoryQuery{
		ChatJID: chatJID,
		Limit:   10000,
	}

	if options != nil && options.DateRange != nil {
		query.After = &options.DateRange.Start
		query.Before = &options.DateRange.End
	}

	// Get messages
	messages, err := e.device.ChatHistory.GetMessages(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Export media files
	for _, msg := range messages {
		if msg.MediaID != nil && *msg.MediaID != "" {
			// Create media file path
			mediaPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s", msg.ID, *msg.MediaID))

			// TODO: Implement actual media file download
			// This would require integration with whatsmeow's media download functionality
			fmt.Printf("Would export media: %s -> %s\n", *msg.MediaID, mediaPath)
		}
	}

	return nil
}
