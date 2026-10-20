-- Chat list (phase 3): pinned chats come first, and the last message keeps its kind and
-- direction so the UI can show a translated preview ("Photo", "You: …").
ALTER TABLE chats ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;
ALTER TABLE chats ADD COLUMN last_message_kind TEXT NOT NULL DEFAULT '';
ALTER TABLE chats ADD COLUMN last_message_from_me INTEGER NOT NULL DEFAULT 0;

DROP INDEX chats_account_recent;
DROP INDEX chats_recent;
CREATE INDEX chats_account_recent ON chats (account_id, pinned DESC, last_message_at DESC);
CREATE INDEX chats_recent ON chats (pinned DESC, last_message_at DESC);
