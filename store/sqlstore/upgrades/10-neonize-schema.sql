-- v9 -> v10: Add neonize-specific tables

-- Message Storage
CREATE TABLE whatsmeow_messages (
    our_jid TEXT,
    chat_jid TEXT,
    sender_jid TEXT,
    message_id TEXT,
    message_type TEXT NOT NULL,
    message_content BYTEA,
    timestamp BIGINT NOT NULL,
    edit_timestamp BIGINT,
    is_edited BOOLEAN DEFAULT FALSE,
    is_revoked BOOLEAN DEFAULT FALSE,
    reply_to_message_id TEXT,
    reply_to_sender_jid TEXT,
    PRIMARY KEY (our_jid, chat_jid, message_id),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE whatsmeow_message_receipts (
    our_jid TEXT,
    chat_jid TEXT,
    message_id TEXT,
    recipient_jid TEXT,
    receipt_type TEXT NOT NULL,
    receipt_timestamp BIGINT NOT NULL,
    PRIMARY KEY (our_jid, chat_jid, message_id, recipient_jid),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Media Storage
CREATE TABLE whatsmeow_media (
    our_jid TEXT,
    media_id TEXT,
    media_type TEXT NOT NULL,
    file_path TEXT,
    file_hash BYTEA NOT NULL,
    media_key BYTEA NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type TEXT,
    upload_timestamp BIGINT NOT NULL,
    PRIMARY KEY (our_jid, media_id),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Enhanced Group Management
CREATE TABLE whatsmeow_group_messages (
    our_jid TEXT,
    group_jid TEXT,
    message_id TEXT,
    sender_jid TEXT,
    message_content BYTEA,
    timestamp BIGINT NOT NULL,
    PRIMARY KEY (our_jid, group_jid, message_id),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE whatsmeow_group_participants (
    our_jid TEXT,
    group_jid TEXT,
    participant_jid TEXT,
    role TEXT NOT NULL,
    join_timestamp BIGINT NOT NULL,
    PRIMARY KEY (our_jid, group_jid, participant_jid),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE whatsmeow_group_invite_links (
    our_jid TEXT,
    group_jid TEXT,
    invite_code TEXT NOT NULL,
    creator_jid TEXT NOT NULL,
    creation_timestamp BIGINT NOT NULL,
    expiration_timestamp BIGINT,
    is_revoked BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (our_jid, group_jid, invite_code),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Newsletter Features
CREATE TABLE whatsmeow_newsletter_subscriptions (
    our_jid TEXT,
    newsletter_jid TEXT,
    subscription_timestamp BIGINT NOT NULL,
    is_muted BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (our_jid, newsletter_jid),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE whatsmeow_newsletter_messages (
    our_jid TEXT,
    newsletter_jid TEXT,
    message_id TEXT,
    message_content BYTEA,
    timestamp BIGINT NOT NULL,
    PRIMARY KEY (our_jid, newsletter_jid, message_id),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Call Management
CREATE TABLE whatsmeow_calls (
    our_jid TEXT,
    call_id TEXT,
    peer_jid TEXT NOT NULL,
    call_type TEXT NOT NULL,
    start_timestamp BIGINT NOT NULL,
    end_timestamp BIGINT,
    call_status TEXT NOT NULL,
    PRIMARY KEY (our_jid, call_id),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
); 