// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chat

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

// AnalyticsPeriod represents a time period for analytics
type AnalyticsPeriod struct {
	Start time.Time
	End   time.Time
}

// MessageAnalytics represents message analytics data
type MessageAnalytics struct {
	TotalMessages      int64
	TotalConversations int64
	Period             AnalyticsPeriod
	MessageTypes       map[string]int64
	SenderStats        map[string]SenderStats
	HourlyActivity     map[int]int64
	DailyActivity      map[string]int64
	WeeklyActivity     map[string]int64
	MonthlyActivity    map[string]int64
	ReactionStats      ReactionStats
	ForwardStats       ForwardStats
	EditStats          EditStats
}

// SenderStats represents statistics for a specific sender
type SenderStats struct {
	MessageCount  int64
	MessageTypes  map[string]int64
	AverageLength float64
	ReactionCount int64
	ForwardCount  int64
	EditCount     int64
	FirstMessage  time.Time
	LastMessage   time.Time
	ActiveDays    int64
}

// ReactionStats represents reaction statistics
type ReactionStats struct {
	TotalReactions int64
	UniqueEmojis   map[string]int64
	TopReactors    map[string]int64
	MostReactedTo  map[string]int64
}

// ForwardStats represents forwarding statistics
type ForwardStats struct {
	TotalForwards  int64
	ForwardSources map[string]int64
	MostForwarded  map[string]int64
}

// EditStats represents message edit statistics
type EditStats struct {
	TotalEdits   int64
	EditRate     float64
	MostEditedBy map[string]int64
}

// ConversationAnalytics represents conversation analytics
type ConversationAnalytics struct {
	ConversationID        types.JID
	MessageCount          int64
	ParticipantCount      int64
	ActiveDays            int64
	FirstMessage          time.Time
	LastMessage           time.Time
	AverageMessagesPerDay float64
	PeakActivityHour      int
	MostActiveDay         string
	MessageTypes          map[string]int64
	TopSenders            []SenderStats
}

// TrendData represents trend analysis data
type TrendData struct {
	Period         AnalyticsPeriod
	DataPoints     []TrendPoint
	TrendDirection string // "increasing", "decreasing", "stable"
	GrowthRate     float64
}

// TrendPoint represents a single data point in a trend
type TrendPoint struct {
	Timestamp time.Time
	Value     float64
}

// MessageAnalyticsEngine provides message analytics functionality
type MessageAnalyticsEngine struct {
	device *store.Device
}

// NewMessageAnalyticsEngine creates a new message analytics engine
func NewMessageAnalyticsEngine(device *store.Device) *MessageAnalyticsEngine {
	return &MessageAnalyticsEngine{
		device: device,
	}
}

// GetMessageAnalytics generates comprehensive message analytics
func (a *MessageAnalyticsEngine) GetMessageAnalytics(ctx context.Context, period *AnalyticsPeriod) (*MessageAnalytics, error) {
	if period == nil {
		// Default to last 30 days
		end := time.Now()
		start := end.AddDate(0, 0, -30)
		period = &AnalyticsPeriod{Start: start, End: end}
	}

	// Build query for all messages in the period
	query := types.ChatHistoryQuery{
		After:  &period.Start,
		Before: &period.End,
		Limit:  100000, // Large limit for analytics
	}

	// Get all messages
	messages, err := a.device.ChatHistory.GetMessages(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Initialize analytics
	analytics := &MessageAnalytics{
		Period:          *period,
		MessageTypes:    make(map[string]int64),
		SenderStats:     make(map[string]SenderStats),
		HourlyActivity:  make(map[int]int64),
		DailyActivity:   make(map[string]int64),
		WeeklyActivity:  make(map[string]int64),
		MonthlyActivity: make(map[string]int64),
		ReactionStats: ReactionStats{
			UniqueEmojis:  make(map[string]int64),
			TopReactors:   make(map[string]int64),
			MostReactedTo: make(map[string]int64),
		},
		ForwardStats: ForwardStats{
			ForwardSources: make(map[string]int64),
			MostForwarded:  make(map[string]int64),
		},
		EditStats: EditStats{
			MostEditedBy: make(map[string]int64),
		},
	}

	// Process messages
	for _, msg := range messages {
		a.processMessageForAnalytics(msg, analytics)
	}

	// Calculate derived statistics
	a.calculateDerivedStats(analytics)

	return analytics, nil
}

// GetConversationAnalytics generates analytics for a specific conversation
func (a *MessageAnalyticsEngine) GetConversationAnalytics(ctx context.Context, chatJID types.JID, period *AnalyticsPeriod) (*ConversationAnalytics, error) {
	if period == nil {
		// Default to last 30 days
		end := time.Now()
		start := end.AddDate(0, 0, -30)
		period = &AnalyticsPeriod{Start: start, End: end}
	}

	// Build query for messages in the conversation
	query := types.ChatHistoryQuery{
		ChatJID: chatJID,
		After:   &period.Start,
		Before:  &period.End,
		Limit:   100000,
	}

	// Get messages
	messages, err := a.device.ChatHistory.GetMessages(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Initialize conversation analytics
	convAnalytics := &ConversationAnalytics{
		ConversationID: chatJID,
		MessageTypes:   make(map[string]int64),
	}

	// Process messages
	senderStats := make(map[string]SenderStats)
	hourlyActivity := make(map[int]int64)
	dailyActivity := make(map[string]int64)
	activeDays := make(map[string]bool)

	for _, msg := range messages {
		convAnalytics.MessageCount++

		// Track message types
		convAnalytics.MessageTypes[msg.Type]++

		// Track sender statistics
		senderKey := msg.SenderJID.String()
		if stats, exists := senderStats[senderKey]; exists {
			stats.MessageCount++
			stats.MessageTypes[msg.Type]++
			if msg.IsEdited {
				stats.EditCount++
			}
			if msg.Timestamp.Before(stats.FirstMessage) {
				stats.FirstMessage = msg.Timestamp
			}
			if msg.Timestamp.After(stats.LastMessage) {
				stats.LastMessage = msg.Timestamp
			}
			senderStats[senderKey] = stats
		} else {
			senderStats[senderKey] = SenderStats{
				MessageCount: 1,
				MessageTypes: map[string]int64{msg.Type: 1},
				FirstMessage: msg.Timestamp,
				LastMessage:  msg.Timestamp,
				EditCount:    map[bool]int64{true: 1, false: 0}[msg.IsEdited],
			}
		}

		// Track activity patterns
		hour := msg.Timestamp.Hour()
		hourlyActivity[hour]++

		day := msg.Timestamp.Format("2006-01-02")
		dailyActivity[day]++
		activeDays[day] = true

		// Track conversation timestamps
		if convAnalytics.FirstMessage.IsZero() || msg.Timestamp.Before(convAnalytics.FirstMessage) {
			convAnalytics.FirstMessage = msg.Timestamp
		}
		if convAnalytics.LastMessage.IsZero() || msg.Timestamp.After(convAnalytics.LastMessage) {
			convAnalytics.LastMessage = msg.Timestamp
		}
	}

	// Calculate derived statistics
	convAnalytics.ActiveDays = int64(len(activeDays))
	if convAnalytics.ActiveDays > 0 {
		convAnalytics.AverageMessagesPerDay = float64(convAnalytics.MessageCount) / float64(convAnalytics.ActiveDays)
	}

	// Find peak activity hour
	var peakHour int
	var peakCount int64
	for hour, count := range hourlyActivity {
		if count > peakCount {
			peakCount = count
			peakHour = hour
		}
	}
	convAnalytics.PeakActivityHour = peakHour

	// Find most active day
	var mostActiveDay string
	var mostActiveCount int64
	for day, count := range dailyActivity {
		if count > mostActiveCount {
			mostActiveCount = count
			mostActiveDay = day
		}
	}
	convAnalytics.MostActiveDay = mostActiveDay

	// Convert sender stats to slice and sort by message count
	for _, stats := range senderStats {
		convAnalytics.TopSenders = append(convAnalytics.TopSenders, stats)
	}
	sort.Slice(convAnalytics.TopSenders, func(i, j int) bool {
		return convAnalytics.TopSenders[i].MessageCount > convAnalytics.TopSenders[j].MessageCount
	})

	return convAnalytics, nil
}

// GetTrendAnalysis analyzes trends in message activity
func (a *MessageAnalyticsEngine) GetTrendAnalysis(ctx context.Context, period *AnalyticsPeriod, granularity string) (*TrendData, error) {
	if period == nil {
		// Default to last 90 days
		end := time.Now()
		start := end.AddDate(0, 0, -90)
		period = &AnalyticsPeriod{Start: start, End: end}
	}

	// Build query
	query := types.ChatHistoryQuery{
		After:  &period.Start,
		Before: &period.End,
		Limit:  100000,
	}

	// Get messages
	messages, err := a.device.ChatHistory.GetMessages(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Group messages by time period
	timeGroups := make(map[string]int64)
	for _, msg := range messages {
		var timeKey string
		switch granularity {
		case "hour":
			timeKey = msg.Timestamp.Format("2006-01-02-15")
		case "day":
			timeKey = msg.Timestamp.Format("2006-01-02")
		case "week":
			year, week := msg.Timestamp.ISOWeek()
			timeKey = fmt.Sprintf("%d-W%02d", year, week)
		case "month":
			timeKey = msg.Timestamp.Format("2006-01")
		default:
			timeKey = msg.Timestamp.Format("2006-01-02")
		}
		timeGroups[timeKey]++
	}

	// Convert to trend points
	var trendPoints []TrendPoint
	for timeKey, count := range timeGroups {
		// Parse time key back to timestamp (simplified)
		var timestamp time.Time
		switch granularity {
		case "hour":
			timestamp, _ = time.Parse("2006-01-02-15", timeKey)
		case "day":
			timestamp, _ = time.Parse("2006-01-02", timeKey)
		case "week":
			// Simplified week parsing
			timestamp, _ = time.Parse("2006-01-02", timeKey+"-01")
		case "month":
			timestamp, _ = time.Parse("2006-01", timeKey)
		default:
			timestamp, _ = time.Parse("2006-01-02", timeKey)
		}

		trendPoints = append(trendPoints, TrendPoint{
			Timestamp: timestamp,
			Value:     float64(count),
		})
	}

	// Sort by timestamp
	sort.Slice(trendPoints, func(i, j int) bool {
		return trendPoints[i].Timestamp.Before(trendPoints[j].Timestamp)
	})

	// Calculate trend direction and growth rate
	trendDirection, growthRate := a.calculateTrendDirection(trendPoints)

	return &TrendData{
		Period:         *period,
		DataPoints:     trendPoints,
		TrendDirection: trendDirection,
		GrowthRate:     growthRate,
	}, nil
}

// GetSenderAnalytics generates analytics for a specific sender
func (a *MessageAnalyticsEngine) GetSenderAnalytics(ctx context.Context, senderJID types.JID, period *AnalyticsPeriod) (*SenderStats, error) {
	if period == nil {
		// Default to last 30 days
		end := time.Now()
		start := end.AddDate(0, 0, -30)
		period = &AnalyticsPeriod{Start: start, End: end}
	}

	// Build query
	query := types.ChatHistoryQuery{
		FromSender: &senderJID,
		After:      &period.Start,
		Before:     &period.End,
		Limit:      100000,
	}

	// Get messages
	messages, err := a.device.ChatHistory.GetMessages(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Initialize sender stats
	stats := &SenderStats{
		MessageTypes: make(map[string]int64),
		FirstMessage: time.Now(),
		LastMessage:  time.Time{},
	}

	// Process messages
	totalLength := 0
	activeDays := make(map[string]bool)

	for _, msg := range messages {
		stats.MessageCount++
		stats.MessageTypes[msg.Type]++

		// Track message length (for text messages)
		if msg.Type == "text" && msg.Message != nil {
			content := a.extractMessageContent(msg)
			totalLength += len(content)
		}

		// Track edit count
		if msg.IsEdited {
			stats.EditCount++
		}

		// Track timestamps
		if msg.Timestamp.Before(stats.FirstMessage) {
			stats.FirstMessage = msg.Timestamp
		}
		if msg.Timestamp.After(stats.LastMessage) {
			stats.LastMessage = msg.Timestamp
		}

		// Track active days
		day := msg.Timestamp.Format("2006-01-02")
		activeDays[day] = true
	}

	// Calculate derived statistics
	stats.ActiveDays = int64(len(activeDays))
	if stats.MessageCount > 0 {
		stats.AverageLength = float64(totalLength) / float64(stats.MessageCount)
	}

	return stats, nil
}

// processMessageForAnalytics processes a single message for analytics
func (a *MessageAnalyticsEngine) processMessageForAnalytics(msg *types.StoredMessage, analytics *MessageAnalytics) {
	analytics.TotalMessages++

	// Track message types
	analytics.MessageTypes[msg.Type]++

	// Track sender statistics
	senderKey := msg.SenderJID.String()
	if stats, exists := analytics.SenderStats[senderKey]; exists {
		stats.MessageCount++
		stats.MessageTypes[msg.Type]++
		if msg.IsEdited {
			stats.EditCount++
		}
		if msg.Timestamp.Before(stats.FirstMessage) {
			stats.FirstMessage = msg.Timestamp
		}
		if msg.Timestamp.After(stats.LastMessage) {
			stats.LastMessage = msg.Timestamp
		}
		analytics.SenderStats[senderKey] = stats
	} else {
		analytics.SenderStats[senderKey] = SenderStats{
			MessageCount: 1,
			MessageTypes: map[string]int64{msg.Type: 1},
			FirstMessage: msg.Timestamp,
			LastMessage:  msg.Timestamp,
			EditCount:    map[bool]int64{true: 1, false: 0}[msg.IsEdited],
		}
	}

	// Track activity patterns
	hour := msg.Timestamp.Hour()
	analytics.HourlyActivity[hour]++

	day := msg.Timestamp.Format("2006-01-02")
	analytics.DailyActivity[day]++

	week := msg.Timestamp.Format("2006-W02")
	analytics.WeeklyActivity[week]++

	month := msg.Timestamp.Format("2006-01")
	analytics.MonthlyActivity[month]++

	// Track edit statistics
	if msg.IsEdited {
		analytics.EditStats.TotalEdits++
		senderKey := msg.SenderJID.String()
		analytics.EditStats.MostEditedBy[senderKey]++
	}
}

// calculateDerivedStats calculates derived statistics
func (a *MessageAnalyticsEngine) calculateDerivedStats(analytics *MessageAnalytics) {
	// Calculate edit rate
	if analytics.TotalMessages > 0 {
		analytics.EditStats.EditRate = float64(analytics.EditStats.TotalEdits) / float64(analytics.TotalMessages) * 100
	}
}

// calculateTrendDirection calculates trend direction and growth rate
func (a *MessageAnalyticsEngine) calculateTrendDirection(points []TrendPoint) (string, float64) {
	if len(points) < 2 {
		return "stable", 0.0
	}

	// Calculate linear regression (simplified)
	var sumX, sumY, sumXY, sumX2 float64
	for i, point := range points {
		x := float64(i)
		y := point.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	n := float64(len(points))
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Determine trend direction
	var direction string
	if slope > 0.1 {
		direction = "increasing"
	} else if slope < -0.1 {
		direction = "decreasing"
	} else {
		direction = "stable"
	}

	return direction, slope
}

// extractMessageContent extracts readable content from a message
func (a *MessageAnalyticsEngine) extractMessageContent(msg *types.StoredMessage) string {
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
