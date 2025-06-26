// Copyright (c) 2025 Novazap Team
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"go.mau.fi/util/dbutil"
)

// SchedulerStore provides methods for managing scheduled messages and templates
type SchedulerStore struct {
	db *dbutil.Database
}

// NewSchedulerStore creates a new SchedulerStore instance
func NewSchedulerStore(db *dbutil.Database) *SchedulerStore {
	return &SchedulerStore{db: db}
}

// MessageTemplate represents a message template for scheduling
type MessageTemplate struct {
	ID               int64
	Name             string
	TemplateType     string
	BaseContent      string
	DynamicFields    json.RawMessage
	ConditionalLogic json.RawMessage
	MediaType        *string
	MediaPath        *string
	Language         string
	Translations     json.RawMessage
	ComplianceTags   json.RawMessage
	TrackingEnabled  bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// MessageDelivery represents a scheduled message delivery
type MessageDelivery struct {
	ID                  int64
	TemplateID          int64
	ScheduleID          *int64
	RecipientType       string
	RecipientID         string
	WhatsAppMessageID   *string
	ScheduledTime       time.Time
	SentTime            *time.Time
	DeliveredTime       *time.Time
	ReadTime            *time.Time
	Status              string
	ErrorMessage        *string
	PersonalizationData json.RawMessage
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Schedule represents a message schedule
type Schedule struct {
	ID          int64
	Title       string
	TemplateID  *int64
	FrequencyID *int64
	GroupID     *int64
	ContactJID  *string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Frequency represents a scheduling frequency
type Frequency struct {
	ID               int64
	Title            string
	Type             string
	IntervalValue    *int
	IntervalUnit     *string
	TimesPerInterval *int
	TimeOfDay        *string
	Status           string
	LastRun          *time.Time
	NextRun          *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ContactGroup represents a group of contacts for scheduling
type ContactGroup struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ContactGroupMember represents a member of a contact group
type ContactGroupMember struct {
	ID         int64
	GroupID    int64
	ContactJID string
	CreatedAt  time.Time
}

// CreateMessageTemplate creates a new message template
func (s *SchedulerStore) CreateMessageTemplate(ctx context.Context, template *MessageTemplate) error {
	query := `
		INSERT INTO sched_message_templates (
			name, template_type, base_content, dynamic_fields, conditional_logic,
			media_type, media_path, language, translations, compliance_tags, tracking_enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(ctx, query,
		template.Name, template.TemplateType, template.BaseContent,
		template.DynamicFields, template.ConditionalLogic, template.MediaType,
		template.MediaPath, template.Language, template.Translations,
		template.ComplianceTags, template.TrackingEnabled)

	if err != nil {
		return fmt.Errorf("failed to create message template: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	template.ID = id
	return nil
}

// GetMessageTemplate retrieves a message template by ID
func (s *SchedulerStore) GetMessageTemplate(ctx context.Context, id int64) (*MessageTemplate, error) {
	query := `
		SELECT id, name, template_type, base_content, dynamic_fields, conditional_logic,
		       media_type, media_path, language, translations, compliance_tags, tracking_enabled,
		       created_at, updated_at
		FROM sched_message_templates WHERE id = ?
	`

	var template MessageTemplate
	err := s.db.QueryRow(ctx, query, id).Scan(
		&template.ID, &template.Name, &template.TemplateType, &template.BaseContent,
		&template.DynamicFields, &template.ConditionalLogic, &template.MediaType,
		&template.MediaPath, &template.Language, &template.Translations,
		&template.ComplianceTags, &template.TrackingEnabled, &template.CreatedAt, &template.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get message template: %w", err)
	}

	return &template, nil
}

// ListMessageTemplates retrieves all message templates
func (s *SchedulerStore) ListMessageTemplates(ctx context.Context) ([]*MessageTemplate, error) {
	query := `
		SELECT id, name, template_type, base_content, dynamic_fields, conditional_logic,
		       media_type, media_path, language, translations, compliance_tags, tracking_enabled,
		       created_at, updated_at
		FROM sched_message_templates ORDER BY created_at DESC
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query message templates: %w", err)
	}
	defer rows.Close()

	var templates []*MessageTemplate
	for rows.Next() {
		var template MessageTemplate
		err := rows.Scan(
			&template.ID, &template.Name, &template.TemplateType, &template.BaseContent,
			&template.DynamicFields, &template.ConditionalLogic, &template.MediaType,
			&template.MediaPath, &template.Language, &template.Translations,
			&template.ComplianceTags, &template.TrackingEnabled, &template.CreatedAt, &template.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan message template: %w", err)
		}
		templates = append(templates, &template)
	}

	return templates, nil
}

// CreateSchedule creates a new schedule
func (s *SchedulerStore) CreateSchedule(ctx context.Context, schedule *Schedule) error {
	query := `
		INSERT INTO sched_schedules (title, template_id, frequency_id, group_id, contact_jid, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(ctx, query,
		schedule.Title, schedule.TemplateID, schedule.FrequencyID,
		schedule.GroupID, schedule.ContactJID, schedule.IsActive)

	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	schedule.ID = id
	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *SchedulerStore) GetSchedule(ctx context.Context, id int64) (*Schedule, error) {
	query := `
		SELECT id, title, template_id, frequency_id, group_id, contact_jid, is_active, created_at, updated_at
		FROM sched_schedules WHERE id = ?
	`

	var schedule Schedule
	err := s.db.QueryRow(ctx, query, id).Scan(
		&schedule.ID, &schedule.Title, &schedule.TemplateID, &schedule.FrequencyID,
		&schedule.GroupID, &schedule.ContactJID, &schedule.IsActive, &schedule.CreatedAt, &schedule.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return &schedule, nil
}

// ListSchedules retrieves all active schedules
func (s *SchedulerStore) ListSchedules(ctx context.Context) ([]*Schedule, error) {
	query := `
		SELECT id, title, template_id, frequency_id, group_id, contact_jid, is_active, created_at, updated_at
		FROM sched_schedules WHERE is_active = true ORDER BY created_at DESC
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*Schedule
	for rows.Next() {
		var schedule Schedule
		err := rows.Scan(
			&schedule.ID, &schedule.Title, &schedule.TemplateID, &schedule.FrequencyID,
			&schedule.GroupID, &schedule.ContactJID, &schedule.IsActive, &schedule.CreatedAt, &schedule.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

// CreateMessageDelivery creates a new message delivery
func (s *SchedulerStore) CreateMessageDelivery(ctx context.Context, delivery *MessageDelivery) error {
	query := `
		INSERT INTO sched_message_deliveries (
			template_id, schedule_id, recipient_type, recipient_id, scheduled_time, status, personalization_data
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(ctx, query,
		delivery.TemplateID, delivery.ScheduleID, delivery.RecipientType,
		delivery.RecipientID, delivery.ScheduledTime, delivery.Status, delivery.PersonalizationData)

	if err != nil {
		return fmt.Errorf("failed to create message delivery: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	delivery.ID = id
	return nil
}

// GetPendingDeliveries retrieves all pending deliveries that should be sent now
func (s *SchedulerStore) GetPendingDeliveries(ctx context.Context) ([]*MessageDelivery, error) {
	query := `
		SELECT id, template_id, schedule_id, recipient_type, recipient_id, whatsapp_message_id,
		       scheduled_time, sent_time, delivered_time, read_time, status, error_message,
		       personalization_data, created_at, updated_at
		FROM sched_message_deliveries 
		WHERE status = 'pending' AND scheduled_time <= ?
		ORDER BY scheduled_time ASC
	`

	rows, err := s.db.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query pending deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*MessageDelivery
	for rows.Next() {
		var delivery MessageDelivery
		err := rows.Scan(
			&delivery.ID, &delivery.TemplateID, &delivery.ScheduleID, &delivery.RecipientType,
			&delivery.RecipientID, &delivery.WhatsAppMessageID, &delivery.ScheduledTime,
			&delivery.SentTime, &delivery.DeliveredTime, &delivery.ReadTime, &delivery.Status,
			&delivery.ErrorMessage, &delivery.PersonalizationData, &delivery.CreatedAt, &delivery.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan delivery: %w", err)
		}
		deliveries = append(deliveries, &delivery)
	}

	return deliveries, nil
}

// UpdateDeliveryStatus updates the status of a message delivery
func (s *SchedulerStore) UpdateDeliveryStatus(ctx context.Context, id int64, status string, whatsappMessageID *string) error {
	query := `
		UPDATE sched_message_deliveries 
		SET status = ?, whatsapp_message_id = ?, sent_time = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := s.db.Exec(ctx, query, status, whatsappMessageID, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to update delivery status: %w", err)
	}

	return nil
}

// CreateContactGroup creates a new contact group
func (s *SchedulerStore) CreateContactGroup(ctx context.Context, group *ContactGroup) error {
	query := `INSERT INTO sched_contact_groups (name) VALUES (?)`

	result, err := s.db.Exec(ctx, query, group.Name)
	if err != nil {
		return fmt.Errorf("failed to create contact group: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	group.ID = id
	return nil
}

// AddContactToGroup adds a contact to a group
func (s *SchedulerStore) AddContactToGroup(ctx context.Context, groupID int64, contactJID string) error {
	query := `INSERT INTO sched_contact_group_members (group_id, contact_jid) VALUES (?, ?)`

	_, err := s.db.Exec(ctx, query, groupID, contactJID)
	if err != nil {
		return fmt.Errorf("failed to add contact to group: %w", err)
	}

	return nil
}

// GetGroupContacts retrieves all contacts in a group
func (s *SchedulerStore) GetGroupContacts(ctx context.Context, groupID int64) ([]string, error) {
	query := `SELECT contact_jid FROM sched_contact_group_members WHERE group_id = ?`

	rows, err := s.db.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query group contacts: %w", err)
	}
	defer rows.Close()

	var contacts []string
	for rows.Next() {
		var contactJID string
		err := rows.Scan(&contactJID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, contactJID)
	}

	return contacts, nil
}

// CreateFrequency creates a new frequency
func (s *SchedulerStore) CreateFrequency(ctx context.Context, frequency *Frequency) error {
	query := `
		INSERT INTO sched_frequencies (
			title, type, interval_value, interval_unit, times_per_interval, time_of_day, status
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(ctx, query,
		frequency.Title, frequency.Type, frequency.IntervalValue, frequency.IntervalUnit,
		frequency.TimesPerInterval, frequency.TimeOfDay, frequency.Status)

	if err != nil {
		return fmt.Errorf("failed to create frequency: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	frequency.ID = id
	return nil
}

// GetFrequency retrieves a frequency by ID
func (s *SchedulerStore) GetFrequency(ctx context.Context, id int64) (*Frequency, error) {
	query := `
		SELECT id, title, type, interval_value, interval_unit, times_per_interval, time_of_day,
		       status, last_run, next_run, created_at, updated_at
		FROM sched_frequencies WHERE id = ?
	`

	var frequency Frequency
	err := s.db.QueryRow(ctx, query, id).Scan(
		&frequency.ID, &frequency.Title, &frequency.Type, &frequency.IntervalValue,
		&frequency.IntervalUnit, &frequency.TimesPerInterval, &frequency.TimeOfDay,
		&frequency.Status, &frequency.LastRun, &frequency.NextRun, &frequency.CreatedAt, &frequency.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get frequency: %w", err)
	}

	return &frequency, nil
}

// UpdateFrequencyNextRun updates the next run time for a frequency
func (s *SchedulerStore) UpdateFrequencyNextRun(ctx context.Context, id int64, nextRun time.Time) error {
	query := `UPDATE sched_frequencies SET next_run = ?, updated_at = ? WHERE id = ?`

	_, err := s.db.Exec(ctx, query, nextRun, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update frequency next run: %w", err)
	}

	return nil
}

// GetSchedulesForFrequency retrieves all schedules that use a specific frequency
func (s *SchedulerStore) GetSchedulesForFrequency(ctx context.Context, frequencyID int64) ([]*Schedule, error) {
	query := `
		SELECT id, title, template_id, frequency_id, group_id, contact_jid, is_active, created_at, updated_at
		FROM sched_schedules WHERE frequency_id = ? AND is_active = true
	`

	rows, err := s.db.Query(ctx, query, frequencyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules for frequency: %w", err)
	}
	defer rows.Close()

	var schedules []*Schedule
	for rows.Next() {
		var schedule Schedule
		err := rows.Scan(
			&schedule.ID, &schedule.Title, &schedule.TemplateID, &schedule.FrequencyID,
			&schedule.GroupID, &schedule.ContactJID, &schedule.IsActive, &schedule.CreatedAt, &schedule.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}
