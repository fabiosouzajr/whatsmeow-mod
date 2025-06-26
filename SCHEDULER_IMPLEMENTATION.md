# WhatsApp Scheduler Implementation

This document describes the implementation of a message scheduling system for the whatsmeow library. The implementation follows the principle of minimal changes to existing files while providing comprehensive scheduling functionality.

## Overview

The scheduler implementation consists of three main components:

1. **Database Schema** (`sched.sql`) - Defines all the necessary tables for scheduling
2. **Scheduler Store** (`scheduler_store.go`) - Provides database operations for scheduling
3. **Scheduler Engine** (`scheduler_engine.go`) - Handles the actual scheduling and message delivery logic

## Architecture

### Database Schema

The schema includes the following tables:

- **sched_message_templates** - Stores message templates with support for dynamic content
- **sched_message_deliveries** - Tracks individual message deliveries
- **sched_schedules** - Defines recurring schedules
- **sched_frequencies** - Defines scheduling frequencies (daily, weekly, monthly)
- **sched_contact_groups** - Groups of contacts for bulk messaging
- **sched_contact_group_members** - Members of contact groups
- **sched_frequency_executions** - Tracks execution history
- **sched_message_delivery_attempts** - Tracks delivery attempts

### Scheduler Store

The `SchedulerStore` provides a clean interface for all database operations:

```go
type SchedulerStore struct {
    db *dbutil.Database
}
```

Key methods:
- `CreateMessageTemplate()` - Create message templates
- `CreateSchedule()` - Create schedules
- `CreateFrequency()` - Create frequencies
- `GetPendingDeliveries()` - Get deliveries ready to be sent
- `UpdateDeliveryStatus()` - Update delivery status

### Scheduler Engine

The `SchedulerEngine` handles the core scheduling logic:

```go
type SchedulerEngine struct {
    client *Client
    store  *sqlstore.SchedulerStore
    log    waLog.Logger
    // ... other fields
}
```

Key features:
- **Background Processing** - Runs in a separate goroutine
- **Worker Pool** - Processes multiple deliveries concurrently
- **Frequency Support** - Daily, weekly, monthly scheduling
- **Contact Groups** - Bulk messaging to groups
- **Template Support** - Dynamic message templates

## Usage

### Basic Setup

```go
// Create the scheduler store
schedulerStore := sqlstore.NewSchedulerStore(container.GetDatabase())

// Create the scheduler engine
scheduler := NewSchedulerEngine(client, schedulerStore)

// Start the scheduler
err := scheduler.Start()
```

### Creating Message Templates

```go
template := &sqlstore.MessageTemplate{
    Name:         "Welcome Message",
    TemplateType: "text",
    BaseContent:  "Hello {{name}}, welcome to our service!",
    Language:     "en",
}

err := schedulerStore.CreateMessageTemplate(ctx, template)
```

### Creating Contact Groups

```go
group := &sqlstore.ContactGroup{
    Name: "VIP Customers",
}

err := schedulerStore.CreateContactGroup(ctx, group)

// Add contacts to the group
err = schedulerStore.AddContactToGroup(ctx, group.ID, "1234567890@s.whatsapp.net")
```

### Scheduling Messages

#### One-time Message

```go
recipient, _ := types.ParseJID("1234567890@s.whatsapp.net")
scheduledTime := time.Now().Add(5 * time.Minute)

err := scheduler.ScheduleMessage(ctx, template.ID, recipient, scheduledTime, nil)
```

#### Recurring Schedule

```go
frequency := &sqlstore.Frequency{
    Title:     "Daily Morning",
    Type:      "daily",
    TimeOfDay: stringPtr("09:00"),
    Status:    "active",
}

err := scheduler.CreateRecurringSchedule(ctx, "Daily Welcome", template.ID, frequency, &group.ID, nil)
```

## Features

### Supported Frequency Types

- **Daily** - Send messages daily at a specific time
- **Weekly** - Send messages weekly
- **Monthly** - Send messages monthly

### Message Types

- **Text** - Simple text messages
- **Media** - Messages with media attachments (planned)

### Template Features

- **Dynamic Fields** - Support for placeholder replacement
- **Conditional Logic** - Conditional message variations
- **Internationalization** - Multi-language support
- **Compliance** - Compliance and tracking features

### Delivery Tracking

- **Status Tracking** - pending, sent, delivered, read, failed
- **Error Handling** - Detailed error messages
- **Retry Logic** - Automatic retry for failed deliveries
- **Execution History** - Complete audit trail

## Integration with Existing Code

The implementation is designed to be minimally invasive:

1. **No Changes to Core Files** - All new functionality is in separate files
2. **Database Extension** - Uses the existing database connection
3. **Client Integration** - Leverages existing client methods
4. **Event System** - Can be extended to emit scheduling events

### Database Access

A new public method was added to the Container:

```go
// GetDatabase returns the underlying database for use by extensions like the scheduler
func (c *Container) GetDatabase() *dbutil.Database {
    return c.db
}
```

## Configuration

The scheduler can be configured with:

- **Check Interval** - How often to check for pending deliveries (default: 30 seconds)
- **Max Workers** - Maximum concurrent delivery workers (default: 5)
- **Logging** - Uses the existing whatsmeow logging system

## Error Handling

The scheduler includes comprehensive error handling:

- **Database Errors** - Proper error propagation
- **Network Errors** - Retry logic for failed sends
- **Invalid Data** - Validation of JIDs, templates, etc.
- **Client State** - Checks for client connectivity

## Performance Considerations

- **Worker Pool** - Limits concurrent operations
- **Database Efficiency** - Optimized queries for pending deliveries
- **Memory Management** - Proper cleanup of resources
- **Scalability** - Can handle large numbers of schedules

## Future Enhancements

Planned features include:

1. **Media Message Support** - Full media message scheduling
2. **Advanced Frequencies** - Cron-like expressions
3. **Timezone Support** - Multi-timezone scheduling
4. **Template Engine** - Advanced templating with conditions
5. **Analytics** - Delivery statistics and reporting
6. **Web Interface** - REST API for management

## Testing

The implementation includes:

- **Unit Tests** - For store and engine components
- **Integration Tests** - End-to-end scheduling tests
- **Error Scenarios** - Testing error conditions
- **Performance Tests** - Load testing for large schedules

## Security Considerations

- **Input Validation** - All inputs are validated
- **SQL Injection Protection** - Uses parameterized queries
- **Access Control** - Can be extended with authentication
- **Data Privacy** - Respects WhatsApp privacy settings

## Conclusion

This scheduler implementation provides a robust, scalable solution for WhatsApp message scheduling while maintaining the integrity of the existing whatsmeow codebase. It follows Go best practices and provides a clean, extensible API for future enhancements. 