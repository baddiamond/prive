-- Key/value application settings (language, theme, window size…).
CREATE TABLE settings (
    key        TEXT    PRIMARY KEY,
    value      TEXT    NOT NULL,
    updated_at INTEGER NOT NULL -- Unix milliseconds
) STRICT;
