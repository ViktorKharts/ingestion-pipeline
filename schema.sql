CREATE TABLE documents (
  id              INTEGER PRIMARY KEY,
  drive_file_id   TEXT NOT NULL,
  filename        TEXT NOT NULL,
  filepath        TEXT NOT NULL,
  content         TEXT NOT NULL,
  extension       TEXT NOT NULL,
  last_modified   TEXT NOT NULL,
  size_bytes      INT NOT NULL
);

CREATE VIRTUAL TABLE documents_fts USING fts5(
    filename,
    content
);

CREATE TRIGGER documents_auto_insert AFTER INSERT ON documents BEGIN
    INSERT INTO documents_fts(rowid, filename, content)
    VALUES (new.id, new.filename, new.content);
END;

CREATE TRIGGER documents_auto_delete AFTER DELETE ON documents BEGIN
    DELETE FROM documents_fts WHERE rowid = old.id;
END;

CREATE TRIGGER documents_auto_update AFTER UPDATE ON documents BEGIN
    UPDATE documents_fts
    SET filename = new.filename, content = new.content
    WHERE rowid = new.id;
END;

