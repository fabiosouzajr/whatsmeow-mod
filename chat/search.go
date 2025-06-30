// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chat

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

// SearchOperator represents search operators
type SearchOperator string

const (
	SearchOperatorAND SearchOperator = "AND"
	SearchOperatorOR  SearchOperator = "OR"
	SearchOperatorNOT SearchOperator = "NOT"
)

// SearchFilter represents a search filter
type SearchFilter struct {
	Field    string
	Operator SearchOperator
	Value    interface{}
}

// AdvancedSearchQuery represents an advanced search query
type AdvancedSearchQuery struct {
	Query       string
	Filters     []SearchFilter
	DateRange   *DateRange
	FromSender  *types.JID
	MessageType *string
	ChatJID     *types.JID
	Limit       int
	Offset      int
	SortBy      string
	SortOrder   string // "asc" or "desc"
}

// SearchResult represents a search result with metadata
type SearchResult struct {
	Message    *types.StoredMessage
	Relevance  float64
	Highlights []string
	MatchType  string
}

// SearchStats represents search statistics
type SearchStats struct {
	TotalResults   int
	SearchTime     time.Duration
	Query          string
	FiltersApplied int
	DateRange      *DateRange
	MessageTypes   map[string]int
	SenderStats    map[string]int
}

// AdvancedMessageSearch provides advanced search functionality
type AdvancedMessageSearch struct {
	device *store.Device
}

// NewAdvancedMessageSearch creates a new advanced message search
func NewAdvancedMessageSearch(device *store.Device) *AdvancedMessageSearch {
	return &AdvancedMessageSearch{
		device: device,
	}
}

// Search performs an advanced search with filters
func (s *AdvancedMessageSearch) Search(ctx context.Context, query *AdvancedSearchQuery) ([]*SearchResult, *SearchStats, error) {
	startTime := time.Now()

	// Build the base query
	baseQuery := types.ChatHistoryQuery{
		Limit: query.Limit,
	}

	// Apply filters
	if query.ChatJID != nil {
		baseQuery.ChatJID = *query.ChatJID
	}

	if query.DateRange != nil {
		baseQuery.After = &query.DateRange.Start
		baseQuery.Before = &query.DateRange.End
	}

	if query.FromSender != nil {
		baseQuery.FromSender = query.FromSender
	}

	if query.MessageType != nil {
		baseQuery.MessageType = query.MessageType
	}

	// Get messages
	messages, err := s.device.ChatHistory.GetMessages(ctx, baseQuery)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Apply search filters
	filteredMessages := s.applyFilters(messages, query.Filters)

	// Apply text search if query is provided
	var searchResults []*SearchResult
	if query.Query != "" {
		searchResults = s.performTextSearch(filteredMessages, query.Query)
	} else {
		// Convert messages to search results
		for _, msg := range filteredMessages {
			searchResults = append(searchResults, &SearchResult{
				Message:   msg,
				Relevance: 1.0,
				MatchType: "filter",
			})
		}
	}

	// Sort results
	s.sortResults(searchResults, query.SortBy, query.SortOrder)

	// Apply pagination
	if query.Offset > 0 && query.Offset < len(searchResults) {
		searchResults = searchResults[query.Offset:]
	}

	if query.Limit > 0 && query.Limit < len(searchResults) {
		searchResults = searchResults[:query.Limit]
	}

	// Calculate statistics
	stats := s.calculateSearchStats(searchResults, query, time.Since(startTime))

	return searchResults, stats, nil
}

// SearchByDateRange searches for messages within a date range
func (s *AdvancedMessageSearch) SearchByDateRange(ctx context.Context, chatJID *types.JID, start, end time.Time, limit int) ([]*SearchResult, error) {
	query := &AdvancedSearchQuery{
		DateRange: &DateRange{Start: start, End: end},
		Limit:     limit,
	}
	if chatJID != nil {
		query.ChatJID = chatJID
	}

	results, _, err := s.Search(ctx, query)
	return results, err
}

// SearchBySender searches for messages from a specific sender
func (s *AdvancedMessageSearch) SearchBySender(ctx context.Context, chatJID *types.JID, sender types.JID, limit int) ([]*SearchResult, error) {
	query := &AdvancedSearchQuery{
		FromSender: &sender,
		Limit:      limit,
	}
	if chatJID != nil {
		query.ChatJID = chatJID
	}

	results, _, err := s.Search(ctx, query)
	return results, err
}

// SearchByMessageType searches for messages of a specific type
func (s *AdvancedMessageSearch) SearchByMessageType(ctx context.Context, chatJID *types.JID, messageType string, limit int) ([]*SearchResult, error) {
	query := &AdvancedSearchQuery{
		MessageType: &messageType,
		Limit:       limit,
	}
	if chatJID != nil {
		query.ChatJID = chatJID
	}

	results, _, err := s.Search(ctx, query)
	return results, err
}

// SearchWithRegex searches for messages using regex patterns
func (s *AdvancedMessageSearch) SearchWithRegex(ctx context.Context, chatJID *types.JID, pattern string, limit int) ([]*SearchResult, error) {
	// Get all messages first (this is simplified - in production you'd want to optimize this)
	baseQuery := types.ChatHistoryQuery{
		Limit: 10000, // Large limit for regex search
	}
	if chatJID != nil {
		baseQuery.ChatJID = *chatJID
	}

	messages, err := s.device.ChatHistory.GetMessages(ctx, baseQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Compile regex pattern
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Search through messages
	var results []*SearchResult
	for _, msg := range messages {
		if s.messageMatchesRegex(msg, regex) {
			results = append(results, &SearchResult{
				Message:   msg,
				Relevance: 1.0,
				MatchType: "regex",
			})
		}

		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

// GetSearchSuggestions provides search suggestions based on partial input
func (s *AdvancedMessageSearch) GetSearchSuggestions(ctx context.Context, partialQuery string, limit int) ([]string, error) {
	// This is a simplified implementation
	// In a real implementation, you'd build a suggestion index

	// Get recent messages to extract common terms
	baseQuery := types.ChatHistoryQuery{
		Limit: 1000,
	}

	messages, err := s.device.ChatHistory.GetMessages(ctx, baseQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Extract words from messages
	wordSet := make(map[string]bool)
	for _, msg := range messages {
		if msg.Message != nil {
			content := s.extractMessageContent(msg)
			words := strings.Fields(strings.ToLower(content))
			for _, word := range words {
				if len(word) > 2 && strings.HasPrefix(word, strings.ToLower(partialQuery)) {
					wordSet[word] = true
				}
			}
		}
	}

	// Convert to slice and limit
	var suggestions []string
	for word := range wordSet {
		suggestions = append(suggestions, word)
		if len(suggestions) >= limit {
			break
		}
	}

	return suggestions, nil
}

// applyFilters applies search filters to messages
func (s *AdvancedMessageSearch) applyFilters(messages []*types.StoredMessage, filters []SearchFilter) []*types.StoredMessage {
	if len(filters) == 0 {
		return messages
	}

	var filtered []*types.StoredMessage
	for _, msg := range messages {
		if s.messagePassesFilters(msg, filters) {
			filtered = append(filtered, msg)
		}
	}

	return filtered
}

// messagePassesFilters checks if a message passes all filters
func (s *AdvancedMessageSearch) messagePassesFilters(msg *types.StoredMessage, filters []SearchFilter) bool {
	for _, filter := range filters {
		if !s.messagePassesFilter(msg, filter) {
			return false
		}
	}
	return true
}

// messagePassesFilter checks if a message passes a specific filter
func (s *AdvancedMessageSearch) messagePassesFilter(msg *types.StoredMessage, filter SearchFilter) bool {
	switch filter.Field {
	case "type":
		return s.compareString(msg.Type, string(filter.Operator), filter.Value.(string))
	case "sender":
		return s.compareString(msg.SenderJID.String(), string(filter.Operator), filter.Value.(string))
	case "is_from_me":
		return s.compareBool(msg.IsFromMe, filter.Operator, filter.Value.(bool))
	case "is_group":
		return s.compareBool(msg.IsGroup, filter.Operator, filter.Value.(bool))
	case "is_edited":
		return s.compareBool(msg.IsEdited, filter.Operator, filter.Value.(bool))
	case "is_revoked":
		return s.compareBool(msg.IsRevoked, filter.Operator, filter.Value.(bool))
	case "status":
		return s.compareString(string(msg.Status), string(filter.Operator), filter.Value.(string))
	case "timestamp":
		if timestamp, ok := filter.Value.(time.Time); ok {
			return s.compareTime(msg.Timestamp, filter.Operator, timestamp)
		}
	}
	return true
}

// compareString compares strings based on operator
func (s *AdvancedMessageSearch) compareString(actual, operator, expected string) bool {
	switch SearchOperator(operator) {
	case SearchOperatorAND:
		return strings.Contains(strings.ToLower(actual), strings.ToLower(expected))
	case SearchOperatorOR:
		return actual == expected
	case SearchOperatorNOT:
		return !strings.Contains(strings.ToLower(actual), strings.ToLower(expected))
	default:
		return actual == expected
	}
}

// compareBool compares booleans based on operator
func (s *AdvancedMessageSearch) compareBool(actual bool, operator SearchOperator, expected bool) bool {
	switch operator {
	case SearchOperatorAND, SearchOperatorOR:
		return actual == expected
	case SearchOperatorNOT:
		return actual != expected
	default:
		return actual == expected
	}
}

// compareTime compares times based on operator
func (s *AdvancedMessageSearch) compareTime(actual time.Time, operator SearchOperator, expected time.Time) bool {
	switch operator {
	case SearchOperatorAND:
		return actual.After(expected)
	case SearchOperatorOR:
		return actual.Before(expected)
	case SearchOperatorNOT:
		return actual.Equal(expected)
	default:
		return actual.Equal(expected)
	}
}

// performTextSearch performs text search on messages
func (s *AdvancedMessageSearch) performTextSearch(messages []*types.StoredMessage, query string) []*SearchResult {
	var results []*SearchResult
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	for _, msg := range messages {
		content := s.extractMessageContent(msg)
		contentLower := strings.ToLower(content)

		// Calculate relevance score
		relevance := s.calculateRelevance(contentLower, queryWords)
		if relevance > 0 {
			highlights := s.extractHighlights(content, queryWords)
			results = append(results, &SearchResult{
				Message:    msg,
				Relevance:  relevance,
				Highlights: highlights,
				MatchType:  "text",
			})
		}
	}

	return results
}

// calculateRelevance calculates relevance score for text search
func (s *AdvancedMessageSearch) calculateRelevance(content string, queryWords []string) float64 {
	if len(queryWords) == 0 {
		return 0
	}

	score := 0.0
	for _, word := range queryWords {
		if strings.Contains(content, word) {
			score += 1.0
		}
	}

	return score / float64(len(queryWords))
}

// extractHighlights extracts highlighted text snippets
func (s *AdvancedMessageSearch) extractHighlights(content string, queryWords []string) []string {
	var highlights []string
	contentLower := strings.ToLower(content)

	for _, word := range queryWords {
		index := strings.Index(contentLower, word)
		if index >= 0 {
			start := max(0, index-20)
			end := min(len(content), index+len(word)+20)
			highlight := content[start:end]
			highlights = append(highlights, highlight)
		}
	}

	return highlights
}

// messageMatchesRegex checks if a message matches a regex pattern
func (s *AdvancedMessageSearch) messageMatchesRegex(msg *types.StoredMessage, regex *regexp.Regexp) bool {
	content := s.extractMessageContent(msg)
	return regex.MatchString(content)
}

// sortResults sorts search results
func (s *AdvancedMessageSearch) sortResults(results []*SearchResult, sortBy, sortOrder string) {
	// This is a simplified implementation
	// In a real implementation, you'd implement proper sorting
	// For now, we'll just sort by relevance
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Relevance < results[j].Relevance {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// calculateSearchStats calculates search statistics
func (s *AdvancedMessageSearch) calculateSearchStats(results []*SearchResult, query *AdvancedSearchQuery, searchTime time.Duration) *SearchStats {
	stats := &SearchStats{
		TotalResults:   len(results),
		SearchTime:     searchTime,
		Query:          query.Query,
		FiltersApplied: len(query.Filters),
		DateRange:      query.DateRange,
		MessageTypes:   make(map[string]int),
		SenderStats:    make(map[string]int),
	}

	// Calculate message type statistics
	for _, result := range results {
		stats.MessageTypes[result.Message.Type]++
		stats.SenderStats[result.Message.SenderJID.String()]++
	}

	return stats
}

// extractMessageContent extracts readable content from a message
func (s *AdvancedMessageSearch) extractMessageContent(msg *types.StoredMessage) string {
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

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
