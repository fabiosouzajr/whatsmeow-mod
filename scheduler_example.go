// Copyright (c) 2025 Novazap Team
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package whatsmeow

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// ExampleSchedulerIntegration demonstrates how to integrate the scheduler with whatsmeow
func ExampleSchedulerIntegration() {
	// Create a database container (this would be your existing whatsmeow setup)
	ctx := context.Background()

	// Example database connection (you would use your actual database)
	container, err := sqlstore.New(ctx, "sqlite3", "file:whatsmeow.db?_foreign_keys=on", waLog.Stdout("Scheduler", "DEBUG", true))
	if err != nil {
		panic(fmt.Sprintf("Failed to create database container: %v", err))
	}

	// Get or create a device
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to get device: %v", err))
	}

	// Create a client
	client := NewClient(device, waLog.Stdout("Client", "DEBUG", true))

	// Connect to WhatsApp
	err = client.Connect()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect: %v", err))
	}

	// Create the scheduler store
	schedulerStore := sqlstore.NewSchedulerStore(container.GetDatabase())

	// Create the scheduler engine
	scheduler := NewSchedulerEngine(client, schedulerStore)

	// Start the scheduler
	err = scheduler.Start()
	if err != nil {
		panic(fmt.Sprintf("Failed to start scheduler: %v", err))
	}

	// Example: Create a message template
	template := &sqlstore.MessageTemplate{
		Name:         "Welcome Message",
		TemplateType: "text",
		BaseContent:  "Hello {{name}}, welcome to our service!",
		Language:     "en",
	}

	err = schedulerStore.CreateMessageTemplate(ctx, template)
	if err != nil {
		panic(fmt.Sprintf("Failed to create template: %v", err))
	}

	// Example: Create a contact group
	group := &sqlstore.ContactGroup{
		Name: "VIP Customers",
	}

	err = schedulerStore.CreateContactGroup(ctx, group)
	if err != nil {
		panic(fmt.Sprintf("Failed to create group: %v", err))
	}

	// Example: Add contacts to the group
	contacts := []string{
		"1234567890@s.whatsapp.net",
		"0987654321@s.whatsapp.net",
	}

	for _, contactJID := range contacts {
		err = schedulerStore.AddContactToGroup(ctx, group.ID, contactJID)
		if err != nil {
			fmt.Printf("Failed to add contact %s to group: %v\n", contactJID, err)
		}
	}

	// Example: Create a frequency for daily messages at 9 AM
	frequency := &sqlstore.Frequency{
		Title:     "Daily Morning",
		Type:      "daily",
		TimeOfDay: stringPtr("09:00"),
		Status:    "active",
	}

	// Example: Create a recurring schedule
	err = scheduler.CreateRecurringSchedule(ctx, "Daily Welcome", template.ID, frequency, &group.ID, nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create recurring schedule: %v", err))
	}

	// Example: Schedule a one-time message
	recipient, _ := types.ParseJID("1234567890@s.whatsapp.net")
	scheduledTime := time.Now().Add(5 * time.Minute) // Send in 5 minutes

	err = scheduler.ScheduleMessage(ctx, template.ID, recipient, scheduledTime, nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to schedule message: %v", err))
	}

	fmt.Println("Scheduler setup complete! Messages will be sent automatically.")

	// Keep the scheduler running
	select {}
}

// ExampleAdvancedScheduling demonstrates more advanced scheduling features
func ExampleAdvancedScheduling() {
	ctx := context.Background()

	// Setup (same as above)
	container, _ := sqlstore.New(ctx, "sqlite3", "file:whatsmeow.db?_foreign_keys=on", waLog.Stdout("Scheduler", "DEBUG", true))
	device, _ := container.GetFirstDevice(ctx)
	client := NewClient(device, waLog.Stdout("Client", "DEBUG", true))
	client.Connect()

	schedulerStore := sqlstore.NewSchedulerStore(container.GetDatabase())
	scheduler := NewSchedulerEngine(client, schedulerStore)
	scheduler.Start()

	// Create different types of templates
	templates := []*sqlstore.MessageTemplate{
		{
			Name:         "Weekly Newsletter",
			TemplateType: "text",
			BaseContent:  "Here's your weekly newsletter with the latest updates!",
		},
		{
			Name:         "Birthday Wish",
			TemplateType: "text",
			BaseContent:  "Happy Birthday {{name}}! 🎉 Hope you have a wonderful day!",
		},
		{
			Name:         "Promotional Image",
			TemplateType: "media",
			BaseContent:  "Check out our latest promotion!",
			MediaType:    stringPtr("image"),
			MediaPath:    stringPtr("/path/to/promotion.jpg"),
		},
	}

	// Create templates
	for _, template := range templates {
		schedulerStore.CreateMessageTemplate(ctx, template)
	}

	// Create different frequencies
	frequencies := []*sqlstore.Frequency{
		{
			Title:     "Weekly",
			Type:      "weekly",
			TimeOfDay: stringPtr("10:00"),
		},
		{
			Title:     "Monthly",
			Type:      "monthly",
			TimeOfDay: stringPtr("15:00"),
		},
		{
			Title:     "Daily Evening",
			Type:      "daily",
			TimeOfDay: stringPtr("18:00"),
		},
	}

	// Create frequencies
	for _, freq := range frequencies {
		schedulerStore.CreateFrequency(ctx, freq)
	}

	// Create contact groups
	groups := []*sqlstore.ContactGroup{
		{Name: "Newsletter Subscribers"},
		{Name: "VIP Customers"},
		{Name: "New Users"},
	}

	for _, group := range groups {
		schedulerStore.CreateContactGroup(ctx, group)
	}

	// Create schedules
	schedules := []struct {
		title      string
		templateID int64
		freqID     int64
		groupID    int64
	}{
		{"Weekly Newsletter", templates[0].ID, frequencies[0].ID, groups[0].ID},
		{"Monthly Promo", templates[2].ID, frequencies[1].ID, groups[1].ID},
		{"Daily Check-in", templates[0].ID, frequencies[2].ID, groups[2].ID},
	}

	for _, sched := range schedules {
		scheduler.CreateRecurringSchedule(ctx, sched.title, sched.templateID,
			&sqlstore.Frequency{ID: sched.freqID}, &sched.groupID, nil)
	}

	fmt.Println("Advanced scheduling setup complete!")
}

// Helper function to create a pointer to a string
func stringPtr(s string) *string {
	return &s
}
