// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chat

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// Example usage of the chat history functionality
func ExampleUsage() {
	// Initialize the whatsmeow client
	client := &whatsmeow.Client{} // Replace with actual client initialization

	// Create chat history client
	chatClient := NewChatHistoryClient(client)

	// Setup the chat history system
	err := chatClient.Setup()
	if err != nil {
		log.Fatalf("Failed to setup chat history: %v", err)
	}

	ctx := context.Background()

	// Example 1: Basic message retrieval
	fmt.Println("=== Basic Message Retrieval ===")
	chatJID, _ := types.ParseJID("1234567890@s.whatsapp.net")
	messages, err := chatClient.GetMessages(ctx, chatJID, &types.ChatHistoryQuery{
		Limit: 10,
	})
	if err != nil {
		log.Printf("Failed to get messages: %v", err)
	} else {
		fmt.Printf("Retrieved %d messages\n", len(messages))
		for _, msg := range messages {
			fmt.Printf("- %s: %s (%s)\n", msg.SenderJID, msg.Type, msg.Timestamp.Format("2006-01-02 15:04:05"))
		}
	}

	// Example 2: Message search
	fmt.Println("\n=== Message Search ===")
	searchResults, err := chatClient.SearchMessages(ctx, chatJID, "hello", 5)
	if err != nil {
		log.Printf("Failed to search messages: %v", err)
	} else {
		fmt.Printf("Found %d messages containing 'hello'\n", len(searchResults))
	}

	// Example 3: Conversation management
	fmt.Println("\n=== Conversation Management ===")
	conversations, err := chatClient.GetConversations(ctx)
	if err != nil {
		log.Printf("Failed to get conversations: %v", err)
	} else {
		fmt.Printf("Found %d conversations\n", len(conversations))
		for _, conv := range conversations {
			fmt.Printf("- %s: %d messages\n", conv.ChatJID, conv.MessageCount)
		}
	}

	// Example 4: Message reactions
	fmt.Println("\n=== Message Reactions ===")
	if len(messages) > 0 {
		reactions, err := chatClient.GetMessageReactions(ctx, chatJID, messages[0].ID)
		if err != nil {
			log.Printf("Failed to get reactions: %v", err)
		} else {
			fmt.Printf("Message has %d reactions\n", len(reactions))
			for _, reaction := range reactions {
				fmt.Printf("- %s: %s\n", reaction.SenderJID, reaction.Emoji)
			}
		}
	}

	// Example 5: Message statistics
	fmt.Println("\n=== Message Statistics ===")
	messageCount, err := chatClient.GetMessageStats(ctx, chatJID)
	if err != nil {
		log.Printf("Failed to get message stats: %v", err)
	} else {
		fmt.Printf("Total messages in chat: %d\n", messageCount)
	}

	// Phase 3: Advanced Features

	// Example 6: Message export
	fmt.Println("\n=== Message Export ===")
	exportOptions := &ExportOptions{
		Format:           ExportFormatJSON,
		IncludeReactions: true,
		IncludeForwards:  true,
		DateRange: &DateRange{
			Start: time.Now().AddDate(0, 0, -7), // Last 7 days
			End:   time.Now(),
		},
	}

	err = chatClient.ExportChatHistory(ctx, chatJID, "chat_export.json", exportOptions)
	if err != nil {
		log.Printf("Failed to export chat history: %v", err)
	} else {
		fmt.Println("Chat history exported successfully")
	}

	// Example 7: Advanced search
	fmt.Println("\n=== Advanced Search ===")
	advancedQuery := &AdvancedSearchQuery{
		Query: "important",
		Filters: []SearchFilter{
			{
				Field:    "type",
				Operator: SearchOperatorAND,
				Value:    "text",
			},
			{
				Field:    "is_from_me",
				Operator: SearchOperatorNOT,
				Value:    true,
			},
		},
		DateRange: &DateRange{
			Start: time.Now().AddDate(0, 0, -30), // Last 30 days
			End:   time.Now(),
		},
		Limit: 20,
	}

	searchResults, stats, err := chatClient.AdvancedSearch(ctx, advancedQuery)
	if err != nil {
		log.Printf("Failed to perform advanced search: %v", err)
	} else {
		fmt.Printf("Advanced search found %d results in %v\n", stats.TotalResults, stats.SearchTime)
		fmt.Printf("Search query: %s\n", stats.Query)
		fmt.Printf("Filters applied: %d\n", stats.FiltersApplied)
	}

	// Example 8: Search by date range
	fmt.Println("\n=== Search by Date Range ===")
	dateResults, err := chatClient.SearchByDateRange(ctx, &chatJID,
		time.Now().AddDate(0, 0, -7), time.Now(), 10)
	if err != nil {
		log.Printf("Failed to search by date range: %v", err)
	} else {
		fmt.Printf("Found %d messages in the last 7 days\n", len(dateResults))
	}

	// Example 9: Search by sender
	fmt.Println("\n=== Search by Sender ===")
	senderJID, _ := types.ParseJID("9876543210@s.whatsapp.net")
	senderResults, err := chatClient.SearchBySender(ctx, &chatJID, senderJID, 10)
	if err != nil {
		log.Printf("Failed to search by sender: %v", err)
	} else {
		fmt.Printf("Found %d messages from %s\n", len(senderResults), senderJID)
	}

	// Example 10: Search by message type
	fmt.Println("\n=== Search by Message Type ===")
	typeResults, err := chatClient.SearchByMessageType(ctx, &chatJID, "image", 10)
	if err != nil {
		log.Printf("Failed to search by message type: %v", err)
	} else {
		fmt.Printf("Found %d image messages\n", len(typeResults))
	}

	// Example 11: Regex search
	fmt.Println("\n=== Regex Search ===")
	regexResults, err := chatClient.SearchWithRegex(ctx, &chatJID, `\b\d{3}-\d{3}-\d{4}\b`, 10)
	if err != nil {
		log.Printf("Failed to perform regex search: %v", err)
	} else {
		fmt.Printf("Found %d messages matching phone number pattern\n", len(regexResults))
	}

	// Example 12: Search suggestions
	fmt.Println("\n=== Search Suggestions ===")
	suggestions, err := chatClient.GetSearchSuggestions(ctx, "hel", 5)
	if err != nil {
		log.Printf("Failed to get search suggestions: %v", err)
	} else {
		fmt.Printf("Search suggestions for 'hel': %v\n", suggestions)
	}

	// Example 13: Message analytics
	fmt.Println("\n=== Message Analytics ===")
	analyticsPeriod := &AnalyticsPeriod{
		Start: time.Now().AddDate(0, 0, -30), // Last 30 days
		End:   time.Now(),
	}

	analytics, err := chatClient.GetMessageAnalytics(ctx, analyticsPeriod)
	if err != nil {
		log.Printf("Failed to get message analytics: %v", err)
	} else {
		fmt.Printf("Analytics for last 30 days:\n")
		fmt.Printf("- Total messages: %d\n", analytics.TotalMessages)
		fmt.Printf("- Message types: %v\n", analytics.MessageTypes)
		fmt.Printf("- Edit rate: %.2f%%\n", analytics.EditStats.EditRate)
	}

	// Example 14: Conversation analytics
	fmt.Println("\n=== Conversation Analytics ===")
	convAnalytics, err := chatClient.GetConversationAnalytics(ctx, chatJID, analyticsPeriod)
	if err != nil {
		log.Printf("Failed to get conversation analytics: %v", err)
	} else {
		fmt.Printf("Conversation analytics:\n")
		fmt.Printf("- Message count: %d\n", convAnalytics.MessageCount)
		fmt.Printf("- Active days: %d\n", convAnalytics.ActiveDays)
		fmt.Printf("- Average messages per day: %.2f\n", convAnalytics.AverageMessagesPerDay)
		fmt.Printf("- Peak activity hour: %d:00\n", convAnalytics.PeakActivityHour)
		fmt.Printf("- Most active day: %s\n", convAnalytics.MostActiveDay)
	}

	// Example 15: Trend analysis
	fmt.Println("\n=== Trend Analysis ===")
	trendData, err := chatClient.GetTrendAnalysis(ctx, analyticsPeriod, "day")
	if err != nil {
		log.Printf("Failed to get trend analysis: %v", err)
	} else {
		fmt.Printf("Trend analysis:\n")
		fmt.Printf("- Trend direction: %s\n", trendData.TrendDirection)
		fmt.Printf("- Growth rate: %.2f\n", trendData.GrowthRate)
		fmt.Printf("- Data points: %d\n", len(trendData.DataPoints))
	}

	// Example 16: Sender analytics
	fmt.Println("\n=== Sender Analytics ===")
	senderAnalytics, err := chatClient.GetSenderAnalytics(ctx, senderJID, analyticsPeriod)
	if err != nil {
		log.Printf("Failed to get sender analytics: %v", err)
	} else {
		fmt.Printf("Sender analytics for %s:\n", senderJID)
		fmt.Printf("- Message count: %d\n", senderAnalytics.MessageCount)
		fmt.Printf("- Average message length: %.2f\n", senderAnalytics.AverageLength)
		fmt.Printf("- Edit count: %d\n", senderAnalytics.EditCount)
		fmt.Printf("- Active days: %d\n", senderAnalytics.ActiveDays)
	}

	// Example 17: Recent activity
	fmt.Println("\n=== Recent Activity ===")
	recentActivity, err := chatClient.GetRecentActivity(ctx, 10)
	if err != nil {
		log.Printf("Failed to get recent activity: %v", err)
	} else {
		fmt.Printf("Recent activity across all conversations:\n")
		for _, msg := range recentActivity {
			fmt.Printf("- %s: %s (%s)\n", msg.ChatJID, msg.SenderJID, msg.Timestamp.Format("2006-01-02 15:04:05"))
		}
	}

	// Example 18: Message threads
	fmt.Println("\n=== Message Threads ===")
	if len(messages) > 0 {
		threadMessages, err := chatClient.GetMessageThread(ctx, chatJID, messages[0].ID, 10)
		if err != nil {
			log.Printf("Failed to get message thread: %v", err)
		} else {
			fmt.Printf("Thread has %d messages\n", len(threadMessages))
		}
	}

	// Example 19: Message replies
	fmt.Println("\n=== Message Replies ===")
	if len(messages) > 0 {
		replies, err := chatClient.GetMessageReplies(ctx, chatJID, messages[0].ID, 10)
		if err != nil {
			log.Printf("Failed to get message replies: %v", err)
		} else {
			fmt.Printf("Message has %d replies\n", len(replies))
		}
	}

	// Example 20: Media messages
	fmt.Println("\n=== Media Messages ===")
	mediaMessages, err := chatClient.GetMediaMessages(ctx, chatJID, nil, 10)
	if err != nil {
		log.Printf("Failed to get media messages: %v", err)
	} else {
		fmt.Printf("Found %d media messages\n", len(mediaMessages))
		for _, msg := range mediaMessages {
			fmt.Printf("- %s: %s\n", msg.Type, *msg.MediaID)
		}
	}

	// Example 21: Message history with pagination
	fmt.Println("\n=== Message History with Pagination ===")
	history, err := chatClient.GetMessageHistory(ctx, chatJID, nil, 5)
	if err != nil {
		log.Printf("Failed to get message history: %v", err)
	} else {
		fmt.Printf("Retrieved %d messages from history\n", len(history))
	}

	// Example 22: Export media files
	fmt.Println("\n=== Export Media Files ===")
	mediaExportOptions := &ExportOptions{
		IncludeMedia: true,
		DateRange: &DateRange{
			Start: time.Now().AddDate(0, 0, -7),
			End:   time.Now(),
		},
	}

	err = chatClient.ExportMedia(ctx, chatJID, "./media_export", mediaExportOptions)
	if err != nil {
		log.Printf("Failed to export media: %v", err)
	} else {
		fmt.Println("Media files exported successfully")
	}

	fmt.Println("\n=== Example completed successfully ===")
}

// Example of setting up the database schema
func ExampleDatabaseSetup() {
	// This would typically be done during application initialization
	fmt.Println("Setting up database schema for chat history...")

	// The schema setup is handled automatically by the SQL store
	// when you initialize the whatsmeow client with the SQL store

	fmt.Println("Database schema setup completed")
}

// Example of error handling
func ExampleErrorHandling() {
	client := &whatsmeow.Client{} // Replace with actual client
	chatClient := NewChatHistoryClient(client)

	ctx := context.Background()
	chatJID, _ := types.ParseJID("1234567890@s.whatsapp.net")

	// Example of handling different types of errors
	messages, err := chatClient.GetMessages(ctx, chatJID, &types.ChatHistoryQuery{
		Limit: 10,
	})

	if err != nil {
		// Handle specific error types
		switch {
		case err.Error() == "conversation not found":
			fmt.Println("Conversation does not exist")
		case err.Error() == "database connection failed":
			fmt.Println("Database connection issue")
		default:
			fmt.Printf("Unexpected error: %v\n", err)
		}
		return
	}

	fmt.Printf("Successfully retrieved %d messages\n", len(messages))
}
