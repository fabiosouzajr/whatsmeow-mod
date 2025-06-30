package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/chat"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func main() {
	// Setup logging
	logger := waLog.Stdout("Main", "DEBUG", true)

	// Initialize database
	db, err := sql.Open("sqlite3", "file:chat_demo.db?_foreign_keys=on")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create SQL store container
	container := sqlstore.NewWithDB(db, "sqlite3", logger)
	if err := container.Upgrade(context.Background()); err != nil {
		log.Fatalf("Failed to upgrade database: %v", err)
	}

	// Check if device exists
	device, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get device: %v", err)
	}

	var client *whatsmeow.Client
	var chatClient *chat.ChatHistoryClient
	connected := false

	if device == nil || device.ID == nil {
		fmt.Println("No device found. Starting pairing process...")
		fmt.Println("Please scan the QR code with your WhatsApp mobile app:")
		fmt.Println()

		// Create a temporary device for pairing
		tempDevice := container.NewDevice()
		tempClient := whatsmeow.NewClient(tempDevice, logger)
		qrChan, _ := tempClient.GetQRChannel(context.Background())
		err = tempClient.Connect()
		if err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}

		// Wait for QR code and connection
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("QR Code:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println()
				fmt.Println("Scan this QR code with your WhatsApp mobile app")
				fmt.Println("(You can also use the URL above in a browser)")
			} else if evt.Event == "success" {
				fmt.Println("QR code scanned! Waiting for connection...")
				break
			} else {
				fmt.Printf("QR event: %s\n", evt.Event)
			}
		}

		// Wait for successful connection
		for i := 0; i < 60; i++ {
			if tempClient.IsConnected() {
				connected = true
				break
			}
			time.Sleep(1 * time.Second)
		}
		if connected {
			fmt.Println("Successfully connected to WhatsApp!")
			// Save the paired device to the DB
			err = tempClient.Store.Save(context.Background())
			if err != nil {
				log.Fatalf("Failed to save paired device: %v", err)
			}
			// Now get the device from the DB
			device, err = container.GetFirstDevice(context.Background())
			if err != nil || device == nil {
				log.Fatalf("Failed to retrieve paired device from DB: %v", err)
			}
			// Ensure the device is properly initialized
			if !device.Initialized {
				container.InitializeDevice(device)
			}
			// Disconnect the temporary client
			tempClient.Disconnect()
		} else {
			log.Fatalf("Failed to connect to WhatsApp after pairing")
		}
	}

	// Now create the real client and chat history client
	client = whatsmeow.NewClient(device, logger)
	chatClient = chat.NewChatHistoryClient(client)
	err = chatClient.Setup()
	if err != nil {
		log.Fatalf("Failed to setup chat history: %v", err)
	}

	// Connect the real client if not already connected
	if !client.IsConnected() {
		err = client.Connect()
		if err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}
		for i := 0; i < 60; i++ {
			if client.IsConnected() {
				connected = true
				break
			}
			time.Sleep(1 * time.Second)
		}
		if connected {
			fmt.Println("Successfully connected to WhatsApp!")
		} else {
			log.Fatalf("Failed to connect to WhatsApp")
		}
	}

	fmt.Println()
	fmt.Println("Chat history system initialized successfully!")
	fmt.Println()

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down...")
		client.Disconnect()
		os.Exit(0)
	}()

	// Only run the demo after a successful connection
	if connected {
		demoChatHistory(chatClient)
	} else {
		fmt.Println("Not connected to WhatsApp. Exiting.")
	}
}

func demoChatHistory(chatClient *chat.ChatHistoryClient) {
	ctx := context.Background()

	fmt.Println("=== Chat History Demo ===")
	fmt.Println("This demo will show you how to use the chat history functionality.")
	fmt.Println("Note: You need to have some messages in your chats to see results.")
	fmt.Println()

	// Get user's own JID for demo
	ownJID := chatClient.GetClient().Store.ID
	if ownJID == nil {
		fmt.Println("Error: Could not get own JID")
		return
	}

	fmt.Printf("Your JID: %s\n", *ownJID)
	fmt.Println()

	// Demo 1: Get conversations
	fmt.Println("1. Getting your conversations...")
	conversations, err := chatClient.GetConversations(ctx)
	if err != nil {
		fmt.Printf("Error getting conversations: %v\n", err)
	} else {
		fmt.Printf("Found %d conversations\n", len(conversations))
		for i, conv := range conversations {
			name := "Unknown"
			if conv.Name != nil {
				name = *conv.Name
			}
			fmt.Printf("  %d. %s (%s) - %d messages\n", i+1, name, conv.ChatJID, conv.MessageCount)
		}
	}
	fmt.Println()

	// Demo 2: Get recent messages from first conversation (if any)
	if len(conversations) > 0 {
		firstChat := conversations[0].ChatJID
		fmt.Printf("2. Getting recent messages from %s...\n", firstChat)
		messages, err := chatClient.GetMessages(ctx, firstChat, &types.ChatHistoryQuery{
			Limit: 5,
		})
		if err != nil {
			fmt.Printf("Error getting messages: %v\n", err)
		} else {
			fmt.Printf("Found %d messages\n", len(messages))
			for i, msg := range messages {
				fmt.Printf("  %d. %s: %s (%s)\n", i+1, msg.SenderJID, msg.Type, msg.Timestamp.Format("2006-01-02 15:04:05"))
			}
		}
		fmt.Println()
	} else {
		fmt.Println("2. No conversations found. Send some messages first!")
		fmt.Println()
	}

	// Demo 3: Search messages (if we have a conversation)
	if len(conversations) > 0 {
		firstChat := conversations[0].ChatJID
		fmt.Printf("3. Searching messages in %s...\n", firstChat)
		searchResults, err := chatClient.SearchMessages(ctx, firstChat, "hello", 3)
		if err != nil {
			fmt.Printf("Error searching messages: %v\n", err)
		} else {
			fmt.Printf("Found %d messages containing 'hello'\n", len(searchResults))
		}
		fmt.Println()
	} else {
		fmt.Println("3. Skipping search demo (no conversations)")
		fmt.Println()
	}

	// Demo 4: Message statistics
	if len(conversations) > 0 {
		firstChat := conversations[0].ChatJID
		fmt.Printf("4. Getting message statistics for %s...\n", firstChat)
		messageCount, err := chatClient.GetMessageStats(ctx, firstChat)
		if err != nil {
			fmt.Printf("Error getting message stats: %v\n", err)
		} else {
			fmt.Printf("Total messages in chat: %d\n", messageCount)
		}
		fmt.Println()
	} else {
		fmt.Println("4. Skipping statistics demo (no conversations)")
		fmt.Println()
	}

	// Demo 5: Advanced search example
	fmt.Println("5. Advanced search example...")
	advancedQuery := &chat.AdvancedSearchQuery{
		Query: "important",
		DateRange: &chat.DateRange{
			Start: time.Now().AddDate(0, 0, -7), // Last 7 days
			End:   time.Now(),
		},
		Limit: 10,
	}

	_, stats, err := chatClient.AdvancedSearch(ctx, advancedQuery)
	if err != nil {
		fmt.Printf("Error performing advanced search: %v\n", err)
	} else {
		fmt.Printf("Advanced search found %d results in %v\n", stats.TotalResults, stats.SearchTime)
		fmt.Printf("Search query: %s\n", stats.Query)
	}
	fmt.Println()

	// Demo 6: Message analytics example
	fmt.Println("6. Message analytics example...")
	analyticsPeriod := &chat.AnalyticsPeriod{
		Start: time.Now().AddDate(0, 0, -30), // Last 30 days
		End:   time.Now(),
	}

	analytics, err := chatClient.GetMessageAnalytics(ctx, analyticsPeriod)
	if err != nil {
		fmt.Printf("Error getting analytics: %v\n", err)
	} else {
		fmt.Printf("Analytics for last 30 days:\n")
		fmt.Printf("  - Total messages: %d\n", analytics.TotalMessages)
		fmt.Printf("  - Message types: %v\n", analytics.MessageTypes)
	}
	fmt.Println()

	// Demo 7: Export example (if we have conversations)
	if len(conversations) > 0 {
		firstChat := conversations[0].ChatJID
		fmt.Printf("7. Export example for %s...\n", firstChat)
		exportOptions := &chat.ExportOptions{
			Format:           chat.ExportFormatJSON,
			IncludeReactions: true,
			IncludeForwards:  true,
			DateRange: &chat.DateRange{
				Start: time.Now().AddDate(0, 0, -7), // Last 7 days
				End:   time.Now(),
			},
		}

		exportFile := fmt.Sprintf("chat_export_%s.json", firstChat.User)
		err = chatClient.ExportChatHistory(ctx, firstChat, exportFile, exportOptions)
		if err != nil {
			fmt.Printf("Error exporting chat history: %v\n", err)
		} else {
			fmt.Printf("Chat history exported successfully to %s\n", exportFile)
		}
		fmt.Println()
	} else {
		fmt.Println("7. Skipping export demo (no conversations)")
		fmt.Println()
	}

	fmt.Println("=== Demo completed! ===")
	fmt.Println("The chat history system is now running and will automatically store:")
	fmt.Println("- New messages you receive")
	fmt.Println("- Message edits and revokes")
	fmt.Println("- Message reactions")
	fmt.Println("- Message forwards")
	fmt.Println()
	fmt.Println("Check the generated files:")
	fmt.Println("- chat_demo.db (SQLite database with your chat history)")
	fmt.Println("- chat_export_*.json (exported chat history files)")
	fmt.Println()
	fmt.Println("Press Ctrl+C to exit...")

	// Keep the program running to maintain the connection
	select {}
}
