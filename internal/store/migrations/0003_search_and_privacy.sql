-- Full-text search over messages (ADR 0009) and the search-exclusion flags it must respect
-- (ADR 0010). Created now so the index never needs a rebuild later.

-- Chats and accounts excluded from search are never indexed.
ALTER TABLE chats ADD COLUMN search_excluded INTEGER NOT NULL DEFAULT 0;
ALTER TABLE accounts ADD COLUMN search_excluded INTEGER NOT NULL DEFAULT 0;

-- Extra searchable text attached to a message: captions, file names, transcripts (ADR 0008).
ALTER TABLE messages ADD COLUMN search_extra TEXT NOT NULL DEFAULT '';

-- Contentless index keyed by messages.rowid: the text already lives in messages, so it is
-- not stored twice. "cancion" finds "canción".
CREATE VIRTUAL TABLE messages_fts USING fts5 (
    text,
    extra,
    content = '',
    contentless_delete = 1,
    tokenize = 'unicode61 remove_diacritics 2'
);

CREATE TRIGGER messages_fts_insert AFTER INSERT ON messages
WHEN NOT EXISTS (SELECT 1 FROM chats WHERE account_id = new.account_id AND id = new.chat_id AND search_excluded = 1)
 AND NOT EXISTS (SELECT 1 FROM accounts WHERE id = new.account_id AND search_excluded = 1)
BEGIN
    INSERT INTO messages_fts (rowid, text, extra) VALUES (new.rowid, new.text, new.search_extra);
END;

CREATE TRIGGER messages_fts_update AFTER UPDATE OF text, search_extra ON messages
BEGIN
    DELETE FROM messages_fts WHERE rowid = old.rowid;
    INSERT INTO messages_fts (rowid, text, extra)
    SELECT new.rowid, new.text, new.search_extra
    WHERE NOT EXISTS (SELECT 1 FROM chats WHERE account_id = new.account_id AND id = new.chat_id AND search_excluded = 1)
      AND NOT EXISTS (SELECT 1 FROM accounts WHERE id = new.account_id AND search_excluded = 1);
END;

CREATE TRIGGER messages_fts_delete AFTER DELETE ON messages
BEGIN
    DELETE FROM messages_fts WHERE rowid = old.rowid;
END;
