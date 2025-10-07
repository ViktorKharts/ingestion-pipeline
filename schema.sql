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

