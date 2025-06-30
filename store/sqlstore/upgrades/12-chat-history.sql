-- v11 -> v12: Add chat history and conversation management tables (SQLite compatible)

-- Enhanced message storage with better indexing and metadata
CREATE TABLE whatsmeow_chat_messages (
    our_jid TEXT,
    chat_jid TEXT,
    message_id TEXT,
    sender_jid TEXT,
    timestamp BIGINT NOT NULL,
    
    -- Message content (serialized protobuf)
    message_content BLOB,
    message_type TEXT NOT NULL,
    
    -- Message metadata
    is_from_me BOOLEAN NOT NULL DEFAULT FALSE,
    is_group BOOLEAN NOT NULL DEFAULT FALSE,
    is_edited BOOLEAN NOT NULL DEFAULT FALSE,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    edit_timestamp BIGINT,
    
    -- Reply information
    reply_to_message_id TEXT,
    reply_to_sender_jid TEXT,
    
    -- Thread information
    thread_message_id TEXT,
    thread_sender_jid TEXT,
    
    -- Message status
    message_status TEXT NOT NULL DEFAULT 'sent',
    
    -- Media information
    media_type TEXT,
    media_id TEXT,
    media_size BIGINT,
    media_mime_type TEXT,
    
    -- Additional metadata
    push_name TEXT,
    verified_name_details BLOB,
    category TEXT,
    multicast BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Search indexing
    search_text TEXT,
    
    PRIMARY KEY (our_jid, chat_jid, message_id),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Indexes for efficient querying
CREATE INDEX idx_chat_messages_chat_timestamp ON whatsmeow_chat_messages(our_jid, chat_jid, timestamp DESC);
CREATE INDEX idx_chat_messages_sender ON whatsmeow_chat_messages(our_jid, chat_jid, sender_jid);
CREATE INDEX idx_chat_messages_type ON whatsmeow_chat_messages(our_jid, chat_jid, message_type);
CREATE INDEX idx_chat_messages_search ON whatsmeow_chat_messages(our_jid, chat_jid) WHERE search_text IS NOT NULL;

-- Full-text search index (PostgreSQL specific, removed for SQLite)
-- CREATE INDEX idx_chat_messages_search_fts ON whatsmeow_chat_messages USING gin(to_tsvector('english', search_text)) WHERE search_text IS NOT NULL;

-- Conversation metadata table
CREATE TABLE whatsmeow_conversations (
    our_jid TEXT,
    chat_jid TEXT,
    
    -- Basic info
    name TEXT,
    description TEXT,
    is_group BOOLEAN NOT NULL DEFAULT FALSE,
    is_broadcast BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Timestamps
    created_at BIGINT NOT NULL,
    last_activity BIGINT NOT NULL,
    last_message BIGINT,
    
    -- Settings
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    muted_until BIGINT,
    
    -- Group-specific fields
    group_invite_link TEXT,
    
    -- Message counts
    message_count BIGINT NOT NULL DEFAULT 0,
    unread_count BIGINT NOT NULL DEFAULT 0,
    
    PRIMARY KEY (our_jid, chat_jid),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Indexes for conversation queries
CREATE INDEX idx_conversations_last_activity ON whatsmeow_conversations(our_jid, last_activity DESC);
CREATE INDEX idx_conversations_archived ON whatsmeow_conversations(our_jid, is_archived, last_activity DESC);
CREATE INDEX idx_conversations_pinned ON whatsmeow_conversations(our_jid, is_pinned DESC, last_activity DESC);

-- Group participants table
CREATE TABLE IF NOT EXISTS whatsmeow_group_participants (
    our_jid TEXT,
    group_jid TEXT,
    participant_jid TEXT,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    join_timestamp BIGINT NOT NULL,
    
    PRIMARY KEY (our_jid, group_jid, participant_jid),
    FOREIGN KEY (our_jid, group_jid) REFERENCES whatsmeow_conversations(our_jid, chat_jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Message reactions table
CREATE TABLE whatsmeow_message_reactions (
    our_jid TEXT,
    chat_jid TEXT,
    message_id TEXT,
    sender_jid TEXT,
    emoji TEXT NOT NULL,
    timestamp BIGINT NOT NULL,
    
    PRIMARY KEY (our_jid, chat_jid, message_id, sender_jid),
    FOREIGN KEY (our_jid, chat_jid, message_id) REFERENCES whatsmeow_chat_messages(our_jid, chat_jid, message_id) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Message forwarding metadata table
CREATE TABLE whatsmeow_message_forwards (
    our_jid TEXT,
    chat_jid TEXT,
    message_id TEXT,
    original_message_id TEXT NOT NULL,
    original_chat_jid TEXT NOT NULL,
    original_sender_jid TEXT NOT NULL,
    original_timestamp BIGINT NOT NULL,
    forward_timestamp BIGINT NOT NULL,
    
    PRIMARY KEY (our_jid, chat_jid, message_id),
    FOREIGN KEY (our_jid, chat_jid, message_id) REFERENCES whatsmeow_chat_messages(our_jid, chat_jid, message_id) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Search index table for better search performance (PostgreSQL only, removed for SQLite)
-- CREATE TABLE whatsmeow_message_search_index (
--     our_jid TEXT,
--     chat_jid TEXT,
--     message_id TEXT,
--     search_vector tsvector,
--     PRIMARY KEY (our_jid, chat_jid, message_id),
--     FOREIGN KEY (our_jid, chat_jid, message_id) REFERENCES whatsmeow_chat_messages(our_jid, chat_jid, message_id) ON DELETE CASCADE ON UPDATE CASCADE
-- );

-- Index for full-text search (PostgreSQL only, removed for SQLite)
-- CREATE INDEX idx_message_search_vector ON whatsmeow_message_search_index USING gin(search_vector);

-- Triggers for automatic search index updates (PostgreSQL only, removed for SQLite)
-- CREATE OR REPLACE FUNCTION update_message_search_index() RETURNS TRIGGER AS $$
-- BEGIN
--     IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
--         INSERT INTO whatsmeow_message_search_index (our_jid, chat_jid, message_id, search_vector)
--         VALUES (NEW.our_jid, NEW.chat_jid, NEW.message_id, to_tsvector('english', COALESCE(NEW.search_text, '')))
--         ON CONFLICT (our_jid, chat_jid, message_id) DO UPDATE SET
--             search_vector = to_tsvector('english', COALESCE(NEW.search_text, ''));
--         RETURN NEW;
--     ELSIF TG_OP = 'DELETE' THEN
--         DELETE FROM whatsmeow_message_search_index WHERE our_jid = OLD.our_jid AND chat_jid = OLD.chat_jid AND message_id = OLD.message_id;
--         RETURN OLD;
--     END IF;
--     RETURN NULL;
-- END;
-- $$ LANGUAGE plpgsql;
--
-- CREATE TRIGGER trigger_update_message_search_index
--     AFTER INSERT OR UPDATE OR DELETE ON whatsmeow_chat_messages
--     FOR EACH ROW EXECUTE FUNCTION update_message_search_index(); 