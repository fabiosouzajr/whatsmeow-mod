// Copyright (c) 2025 Novazap Team
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package whatsmeow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// SchedulerEngine handles the scheduling and execution of messages
type SchedulerEngine struct {
	client *Client
	store  *sqlstore.SchedulerStore
	log    waLog.Logger

	// Scheduler state
	running  bool
	stopChan chan struct{}
	workerWg sync.WaitGroup

	// Configuration
	checkInterval time.Duration
	maxWorkers    int
}

// NewSchedulerEngine creates a new scheduler engine
func NewSchedulerEngine(client *Client, store *sqlstore.SchedulerStore) *SchedulerEngine {
	return &SchedulerEngine{
		client:        client,
		store:         store,
		log:           client.Log,
		stopChan:      make(chan struct{}),
		checkInterval: 30 * time.Second, // Check every 30 seconds
		maxWorkers:    5,                // Max 5 concurrent workers
	}
}

// Start starts the scheduler engine
func (se *SchedulerEngine) Start() error {
	if se.running {
		return fmt.Errorf("scheduler engine is already running")
	}

	se.running = true
	se.log.Infof("Starting scheduler engine with check interval: %v", se.checkInterval)

	// Start the main scheduler loop
	se.workerWg.Add(1)
	go se.schedulerLoop()

	return nil
}

// Stop stops the scheduler engine
func (se *SchedulerEngine) Stop() error {
	if !se.running {
		return fmt.Errorf("scheduler engine is not running")
	}

	se.log.Infof("Stopping scheduler engine...")
	se.running = false
	close(se.stopChan)
	se.workerWg.Wait()
	se.log.Infof("Scheduler engine stopped")

	return nil
}

// schedulerLoop is the main loop that checks for pending deliveries
func (se *SchedulerEngine) schedulerLoop() {
	defer se.workerWg.Done()

	ticker := time.NewTicker(se.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-se.stopChan:
			return
		case <-ticker.C:
			se.processPendingDeliveries()
		}
	}
}

// processPendingDeliveries processes all pending message deliveries
func (se *SchedulerEngine) processPendingDeliveries() {
	ctx := context.Background()

	// Get pending deliveries
	deliveries, err := se.store.GetPendingDeliveries(ctx)
	if err != nil {
		se.log.Errorf("Failed to get pending deliveries: %v", err)
		return
	}

	if len(deliveries) == 0 {
		return
	}

	se.log.Infof("Processing %d pending deliveries", len(deliveries))

	// Process deliveries with worker pool
	semaphore := make(chan struct{}, se.maxWorkers)
	var wg sync.WaitGroup

	for _, delivery := range deliveries {
		wg.Add(1)
		go func(d *sqlstore.MessageDelivery) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire worker slot
			defer func() { <-semaphore }() // Release worker slot

			se.processDelivery(ctx, d)
		}(delivery)
	}

	wg.Wait()
}

// processDelivery processes a single message delivery
func (se *SchedulerEngine) processDelivery(ctx context.Context, delivery *sqlstore.MessageDelivery) {
	se.log.Debugf("Processing delivery %d for template %d", delivery.ID, delivery.TemplateID)

	// Get the message template
	template, err := se.store.GetMessageTemplate(ctx, delivery.TemplateID)
	if err != nil {
		se.log.Errorf("Failed to get template %d for delivery %d: %v", delivery.TemplateID, delivery.ID, err)
		errorMsg := fmt.Sprintf("Template not found: %v", err)
		se.updateDeliveryStatus(ctx, delivery.ID, "failed", nil, &errorMsg)
		return
	}

	if template == nil {
		se.log.Errorf("Template %d not found for delivery %d", delivery.TemplateID, delivery.ID)
		errorMsg := "Template not found"
		se.updateDeliveryStatus(ctx, delivery.ID, "failed", nil, &errorMsg)
		return
	}

	// Parse recipient JID
	recipientJID, err := types.ParseJID(delivery.RecipientID)
	if err != nil {
		se.log.Errorf("Invalid recipient JID %s for delivery %d: %v", delivery.RecipientID, delivery.ID, err)
		errorMsg := fmt.Sprintf("Invalid JID: %v", err)
		se.updateDeliveryStatus(ctx, delivery.ID, "failed", nil, &errorMsg)
		return
	}

	// Prepare message content
	messageContent := se.prepareMessageContent(template, delivery.PersonalizationData)

	// Send the message
	messageID, err := se.sendScheduledMessage(ctx, recipientJID, messageContent, template)
	if err != nil {
		se.log.Errorf("Failed to send scheduled message for delivery %d: %v", delivery.ID, err)
		errorMsg := err.Error()
		se.updateDeliveryStatus(ctx, delivery.ID, "failed", nil, &errorMsg)
		return
	}

	// Update delivery status
	se.updateDeliveryStatus(ctx, delivery.ID, "sent", &messageID, nil)

	se.log.Infof("Successfully sent scheduled message %s for delivery %d", messageID, delivery.ID)
}

// prepareMessageContent prepares the message content from template and personalization data
func (se *SchedulerEngine) prepareMessageContent(template *sqlstore.MessageTemplate, personalizationData []byte) string {
	// For now, just return the base content
	// TODO: Implement dynamic field replacement and conditional logic
	return template.BaseContent
}

// sendScheduledMessage sends a scheduled message
func (se *SchedulerEngine) sendScheduledMessage(ctx context.Context, recipient types.JID, content string, template *sqlstore.MessageTemplate) (string, error) {
	// Check if client is connected
	if !se.client.IsConnected() {
		return "", fmt.Errorf("client is not connected")
	}

	// Send the message based on template type
	switch template.TemplateType {
	case "text":
		return se.sendTextMessage(ctx, recipient, content)
	case "media":
		return se.sendMediaMessage(ctx, recipient, content, template)
	default:
		return se.sendTextMessage(ctx, recipient, content)
	}
}

// sendTextMessage sends a text message
func (se *SchedulerEngine) sendTextMessage(ctx context.Context, recipient types.JID, content string) (string, error) {
	// Use the client's SendMessage method with proper message structure
	msg := &waE2E.Message{
		Conversation: proto.String(content),
	}

	resp, err := se.client.SendMessage(ctx, recipient, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send text message: %w", err)
	}

	return string(resp.ID), nil
}

// sendMediaMessage sends a media message
func (se *SchedulerEngine) sendMediaMessage(ctx context.Context, recipient types.JID, content string, template *sqlstore.MessageTemplate) (string, error) {
	if template.MediaPath == nil {
		return "", fmt.Errorf("media path not specified for media template")
	}

	// TODO: Implement media message sending
	// This would involve uploading the media file and sending it with the content as caption
	se.log.Warnf("Media message sending not yet implemented for delivery to %s", recipient)

	// For now, just send as text message
	return se.sendTextMessage(ctx, recipient, content)
}

// updateDeliveryStatus updates the status of a delivery
func (se *SchedulerEngine) updateDeliveryStatus(ctx context.Context, deliveryID int64, status string, messageID *string, errorMsg *string) {
	err := se.store.UpdateDeliveryStatus(ctx, deliveryID, status, messageID)
	if err != nil {
		se.log.Errorf("Failed to update delivery status for delivery %d: %v", deliveryID, err)
	}
}

// ScheduleMessage schedules a message for delivery
func (se *SchedulerEngine) ScheduleMessage(ctx context.Context, templateID int64, recipient types.JID, scheduledTime time.Time, personalizationData []byte) error {
	delivery := &sqlstore.MessageDelivery{
		TemplateID:          templateID,
		RecipientType:       "contact",
		RecipientID:         recipient.String(),
		ScheduledTime:       scheduledTime,
		Status:              "pending",
		PersonalizationData: personalizationData,
	}

	return se.store.CreateMessageDelivery(ctx, delivery)
}

// ScheduleGroupMessage schedules a message for delivery to a group
func (se *SchedulerEngine) ScheduleGroupMessage(ctx context.Context, templateID int64, groupID int64, scheduledTime time.Time, personalizationData []byte) error {
	// Get all contacts in the group
	contacts, err := se.store.GetGroupContacts(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to get group contacts: %w", err)
	}

	// Schedule message for each contact
	for _, contactJID := range contacts {
		recipient, err := types.ParseJID(contactJID)
		if err != nil {
			se.log.Warnf("Invalid contact JID %s in group %d: %v", contactJID, groupID, err)
			continue
		}

		err = se.ScheduleMessage(ctx, templateID, recipient, scheduledTime, personalizationData)
		if err != nil {
			se.log.Errorf("Failed to schedule message for contact %s: %v", contactJID, err)
		}
	}

	return nil
}

// CreateRecurringSchedule creates a recurring schedule
func (se *SchedulerEngine) CreateRecurringSchedule(ctx context.Context, title string, templateID int64, frequency *sqlstore.Frequency, groupID *int64, contactJID *string) error {
	// Create the frequency first
	err := se.store.CreateFrequency(ctx, frequency)
	if err != nil {
		return fmt.Errorf("failed to create frequency: %w", err)
	}

	// Create the schedule
	schedule := &sqlstore.Schedule{
		Title:       title,
		TemplateID:  &templateID,
		FrequencyID: &frequency.ID,
		GroupID:     groupID,
		ContactJID:  contactJID,
		IsActive:    true,
	}

	err = se.store.CreateSchedule(ctx, schedule)
	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	// Calculate next run time and create initial deliveries
	err = se.createInitialDeliveries(ctx, schedule, frequency)
	if err != nil {
		return fmt.Errorf("failed to create initial deliveries: %w", err)
	}

	return nil
}

// createInitialDeliveries creates the initial message deliveries for a schedule
func (se *SchedulerEngine) createInitialDeliveries(ctx context.Context, schedule *sqlstore.Schedule, frequency *sqlstore.Frequency) error {
	// Calculate next run time based on frequency
	nextRun := se.calculateNextRunTime(frequency)
	if nextRun.IsZero() {
		return fmt.Errorf("could not calculate next run time for frequency")
	}

	// Update frequency with next run time
	err := se.store.UpdateFrequencyNextRun(ctx, frequency.ID, nextRun)
	if err != nil {
		return fmt.Errorf("failed to update frequency next run: %w", err)
	}

	// Create deliveries based on schedule type
	if schedule.GroupID != nil {
		// Schedule for group
		return se.ScheduleGroupMessage(ctx, *schedule.TemplateID, *schedule.GroupID, nextRun, nil)
	} else if schedule.ContactJID != nil {
		// Schedule for single contact
		recipient, err := types.ParseJID(*schedule.ContactJID)
		if err != nil {
			return fmt.Errorf("invalid contact JID: %w", err)
		}
		return se.ScheduleMessage(ctx, *schedule.TemplateID, recipient, nextRun, nil)
	}

	return fmt.Errorf("schedule must have either group_id or contact_jid")
}

// calculateNextRunTime calculates the next run time based on frequency
func (se *SchedulerEngine) calculateNextRunTime(frequency *sqlstore.Frequency) time.Time {
	now := time.Now()

	switch frequency.Type {
	case "daily":
		if frequency.TimeOfDay != nil {
			// Parse time of day (e.g., "09:00")
			t, err := time.Parse("15:04", *frequency.TimeOfDay)
			if err != nil {
				se.log.Errorf("Invalid time of day format: %s", *frequency.TimeOfDay)
				return time.Time{}
			}

			next := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
			if next.Before(now) {
				next = next.Add(24 * time.Hour)
			}
			return next
		}
		return now.Add(24 * time.Hour)

	case "weekly":
		interval := 1
		if frequency.IntervalValue != nil {
			interval = *frequency.IntervalValue
		}
		return now.AddDate(0, 0, 7*interval)

	case "monthly":
		interval := 1
		if frequency.IntervalValue != nil {
			interval = *frequency.IntervalValue
		}
		return now.AddDate(0, interval, 0)

	default:
		se.log.Warnf("Unsupported frequency type: %s", frequency.Type)
		return time.Time{}
	}
}

// IsRunning returns whether the scheduler engine is running
func (se *SchedulerEngine) IsRunning() bool {
	return se.running
}
