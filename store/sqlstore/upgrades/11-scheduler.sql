-- v10 -> v11: Add scheduler-specific tables
-- Enhanced Message Templates
-- This table allows for creating and managing message templates with advanced features
CREATE TABLE IF NOT EXISTS sched_message_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    template_type TEXT NOT NULL CHECK(template_type IN (
        'text', 
        'media', 
        'rich_text', 
        'dynamic', 
        'conditional'
    )),

    -- Core Message Content
    base_content TEXT NOT NULL,
    
    -- Dynamic Templating Support
    dynamic_fields JSON,        -- Placeholders for personalization
    conditional_logic JSON,     -- Conditional message variations
    
    -- Media Support
    media_type TEXT,            -- image, video, document
    media_path TEXT,
    
    -- Internationalization
    language TEXT DEFAULT 'en',
    translations JSON,
    
    -- Compliance & Tracking
    compliance_tags JSON,
    tracking_enabled BOOLEAN DEFAULT FALSE,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Message Deliveries Table
CREATE TABLE IF NOT EXISTS sched_message_deliveries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id INTEGER NOT NULL,
    schedule_id INTEGER,
    recipient_type TEXT NOT NULL CHECK(recipient_type IN ('contact', 'group')),
    recipient_id TEXT NOT NULL,
    whatsapp_message_id TEXT,
    scheduled_time DATETIME NOT NULL,
    sent_time DATETIME,
    delivered_time DATETIME,
    read_time DATETIME,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'sent', 'delivered', 'read', 'failed')),
    error_message TEXT,
    personalization_data JSON,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (template_id) REFERENCES sched_message_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (schedule_id) REFERENCES sched_schedules(id) ON DELETE SET NULL
);

-- Template Personalization Mapping
-- This table allows for mapping personalization fields to template placeholders
CREATE TABLE IF NOT EXISTS sched_template_personalization (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id INTEGER,
    contact_field TEXT,         -- Field to pull personalization from
    placeholder TEXT,           -- Placeholder in template
    FOREIGN KEY (template_id) REFERENCES sched_message_templates(id)
);

-- Schedules
-- This table stores the scheduling rules for each schedule
CREATE TABLE IF NOT EXISTS sched_schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL UNIQUE,
    template_id INTEGER,
    frequency_id INTEGER,
    group_id INTEGER,
    contact_jid TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (template_id) REFERENCES sched_message_templates(id) ON DELETE SET NULL,
    FOREIGN KEY (frequency_id) REFERENCES sched_frequencies(id) ON DELETE SET NULL,
    FOREIGN KEY (group_id) REFERENCES sched_contact_groups(id) ON DELETE SET NULL
);

-- Create message_delivery_attempts table
CREATE TABLE IF NOT EXISTS sched_message_delivery_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    delivery_id INTEGER NOT NULL,
    attempt_time TIMESTAMP NOT NULL,
    status TEXT CHECK (status IN ('success', 'failed')) NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (delivery_id) REFERENCES sched_message_deliveries(id) ON DELETE CASCADE
);

-- ------------------------------------------------------------
-- Frequency Tables
-- ------------------------------------------------------------

-- Create frequencies table
CREATE TABLE IF NOT EXISTS sched_frequencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    interval_value INTEGER,
    interval_unit TEXT,
    times_per_interval INTEGER,
    time_of_day TEXT,
    status TEXT DEFAULT 'active',
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Advanced frequencies table
-- This table allows for more advanced scheduling options, including:
-- - Simple recurring frequencies (daily, weekly, monthly, etc.)
-- - Complex recurring frequencies with specific days and time constraints
-- - Custom expressions for cron-like scheduling
-- - Exclusion rules for specific dates or holidays
-- - Timezone support for scheduling across different time zones
-- - Flexible execution parameters such as jitter and retry strategies
-- - Status tracking for active, paused, or expired frequencies
-- - Creation and update timestamps for auditing purposes
CREATE TABLE IF NOT EXISTS sched_advanced_frequencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL CHECK(type IN (
        'simple_recurring',   -- Daily/Weekly/Monthly
        'complex_recurring',  -- More advanced patterns
        'custom_expression'   -- Cron-like expressions
    )),
    
    -- Simple Recurring Parameters
    interval_type TEXT CHECK(interval_type IN (
        'daily', 'weekly', 'monthly', 
        'yearly', 'hourly', 'custom'
    )),
    interval_value INTEGER DEFAULT 1,
    
    -- Complex Recurring Parameters
    specific_days JSON,        -- Specific days of week/month
    time_constraints JSON,     -- Time windows
    
    -- Advanced Scheduling Parameters
    start_date DATETIME,
    end_date DATETIME,
    max_occurrences INTEGER,
    
    -- Custom Expression (Cron-like)
    cron_expression TEXT,
    
    -- Exclusion Rules
    exclusion_dates JSON,      -- Dates to skip
    holiday_handling BOOLEAN DEFAULT FALSE,
    
    -- Timezone Support
    timezone TEXT DEFAULT 'UTC',
    
    -- Flexible Execution Parameters
    jitter_minutes INTEGER DEFAULT 0,  -- Random delay
    retry_strategy JSON,
    
    status TEXT DEFAULT 'active' CHECK(status IN ('active', 'paused', 'expired')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Advanced Scheduling Rules Table
CREATE TABLE IF NOT EXISTS sched_scheduling_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    frequency_id INTEGER,
    rule_type TEXT CHECK(rule_type IN (
        'time_window',     -- Specific time ranges
        'day_restriction', -- Limit to specific days
        'load_balancing',  -- Spread messages
        'priority'         -- Message priority
    )),
    rule_configuration JSON,
    FOREIGN KEY (frequency_id) REFERENCES sched_advanced_frequencies(id)
);


-- Create frequency_week_days table
-- This table allows for specifying which days of the week a frequency applies to
-- It links to the sched_frequencies table and ensures that a frequency can have multiple days
-- It includes a unique constraint to prevent duplicate entries for the same frequency and day
-- The day is stored as a text value (e.g., 'Monday', 'Tuesday', etc.)
-- The created_at timestamp is automatically set to the current time when a new entry is created
-- The frequency_id is a foreign key that references the id in the sched_frequencies table
CREATE TABLE IF NOT EXISTS sched_frequency_week_days (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    frequency_id INTEGER NOT NULL,
    day TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (frequency_id) REFERENCES sched_frequencies(id) ON DELETE CASCADE,
    UNIQUE(frequency_id, day)
);



-- ------------------------------------------------------------
-- Contact Groups
-- ------------------------------------------------------------

-- Create contact_groups table
CREATE TABLE IF NOT EXISTS sched_contact_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create contact_group_members table
CREATE TABLE IF NOT EXISTS sched_contact_group_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    contact_jid TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES sched_contact_groups(id) ON DELETE CASCADE,
    UNIQUE(group_id, contact_jid)
);

-- Add contact filtering capabilities
-- This table allows for filtering contacts based on tags, statuses, or custom criteria
-- It links to the sched_contact_groups table and includes a filter type and value
CREATE TABLE IF NOT EXISTS sched_contact_filters (
    id INTEGER PRIMARY KEY,
    group_id INTEGER,
    filter_type TEXT CHECK(filter_type IN ('tag', 'status', 'custom')),
    filter_value TEXT,
    FOREIGN KEY (group_id) REFERENCES sched_contact_groups(id)
);


-- ------------------------------------------------------------
-- Scheduling Tables
-- ------------------------------------------------------------

-- Create frequency_recurrence_rules table
-- This table stores the recurrence rules for each frequency
-- It is used to determine the next run time for each frequency
CREATE TABLE IF NOT EXISTS sched_frequency_recurrence_rules (
    frequency_id INTEGER,
    rule_type TEXT CHECK (rule_type IN ('exclude', 'include')),
    rule_date DATE,
    FOREIGN KEY (frequency_id) REFERENCES sched_frequencies(id) ON DELETE CASCADE
);

-- Create frequency_executions table
-- This table stores the execution history for each frequency
-- It is used to track the status of each execution
CREATE TABLE IF NOT EXISTS sched_frequency_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    frequency_id INTEGER,
    scheduled_time TIMESTAMP,
    actual_time TIMESTAMP,
    status TEXT CHECK (status IN ('success', 'failed', 'skipped')),
    error_message TEXT,
    FOREIGN KEY (frequency_id) REFERENCES sched_frequencies(id) ON DELETE CASCADE
);

