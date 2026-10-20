-- One row per connected account. Times are Unix milliseconds.
CREATE TABLE accounts (
    id           TEXT    PRIMARY KEY,
    engine_id    TEXT    NOT NULL,
    remote_id    TEXT    NOT NULL DEFAULT '',
    display_name TEXT    NOT NULL DEFAULT '',
    color        TEXT    NOT NULL DEFAULT '',
    position     INTEGER NOT NULL DEFAULT 0,
    created_at   INTEGER NOT NULL
) STRICT;

CREATE TABLE chats (
    account_id           TEXT    NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    id                   TEXT    NOT NULL,
    name                 TEXT    NOT NULL DEFAULT '',
    kind                 TEXT    NOT NULL DEFAULT 'direct',
    unread_count         INTEGER NOT NULL DEFAULT 0,
    last_message_preview TEXT    NOT NULL DEFAULT '',
    last_message_at      INTEGER NOT NULL DEFAULT 0, -- 0 = no messages
    muted                INTEGER NOT NULL DEFAULT 0,
    archived             INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (account_id, id)
) STRICT;

-- Per-account inbox and the unified inbox, newest first.
CREATE INDEX chats_account_recent ON chats (account_id, last_message_at DESC);
CREATE INDEX chats_recent ON chats (last_message_at DESC);

-- Messages may arrive before their chat, so they reference the account only.
CREATE TABLE messages (
    account_id  TEXT    NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    chat_id     TEXT    NOT NULL,
    id          TEXT    NOT NULL,
    sender_id   TEXT    NOT NULL DEFAULT '',
    sender_name TEXT    NOT NULL DEFAULT '',
    from_me     INTEGER NOT NULL DEFAULT 0,
    timestamp   INTEGER NOT NULL,
    kind        TEXT    NOT NULL DEFAULT 'text',
    text        TEXT    NOT NULL DEFAULT '',
    status      TEXT    NOT NULL DEFAULT '',
    reply_to    TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (account_id, chat_id, id)
) STRICT;

-- History pagination within a chat.
CREATE INDEX messages_chat_time ON messages (account_id, chat_id, timestamp DESC);
