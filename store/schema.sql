-- todos table
CREATE TABLE IF NOT EXISTS todos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  priority TEXT CHECK (length(priority) = 1) DEFAULT NULL,
  completed INTEGER NOT NULL DEFAULT 0,
  archived  INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  due_at     TEXT DEFAULT NULL,
  threshold_at TEXT DEFAULT NULL,
  recurrence_rule TEXT DEFAULT NULL,
  source_line TEXT,
  notes_md TEXT DEFAULT '',
  notes_text TEXT DEFAULT '',
  section TEXT DEFAULT 'anytime',
  pinned INTEGER NOT NULL DEFAULT 0,
  type TEXT NOT NULL DEFAULT 'todo',
  inbox INTEGER NOT NULL DEFAULT 0,
  api_source TEXT DEFAULT NULL
);

CREATE TRIGGER IF NOT EXISTS todos_update_ts
AFTER UPDATE ON todos FOR EACH ROW
BEGIN
  UPDATE todos SET updated_at = datetime('now') WHERE id = NEW.id;
END;

-- taxonomy tables
CREATE TABLE IF NOT EXISTS projects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS contexts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS tags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);

-- link tables
CREATE TABLE IF NOT EXISTS todo_projects (
  todo_id INTEGER, project_id INTEGER,
  PRIMARY KEY (todo_id, project_id),
  FOREIGN KEY (todo_id) REFERENCES todos(id) ON DELETE CASCADE,
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS todo_contexts (
  todo_id INTEGER, context_id INTEGER,
  PRIMARY KEY (todo_id, context_id),
  FOREIGN KEY (todo_id) REFERENCES todos(id) ON DELETE CASCADE,
  FOREIGN KEY (context_id) REFERENCES contexts(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS todo_tags (
  todo_id INTEGER, tag_id INTEGER,
  PRIMARY KEY (todo_id, tag_id),
  FOREIGN KEY (todo_id) REFERENCES todos(id) ON DELETE CASCADE,
  FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- references between todos/notes
CREATE TABLE IF NOT EXISTS refs (
  source_id INTEGER NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
  target_id INTEGER NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
  PRIMARY KEY (source_id, target_id)
);

-- FTS5
CREATE VIRTUAL TABLE IF NOT EXISTS todos_fts USING fts5(
  title, notes_text, content='todos', content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS todos_ai AFTER INSERT ON todos BEGIN
  INSERT INTO todos_fts(rowid, title, notes_text) VALUES (new.id, new.title, new.notes_text);
END;
CREATE TRIGGER IF NOT EXISTS todos_ad AFTER DELETE ON todos BEGIN
  INSERT INTO todos_fts(todos_fts, rowid, title, notes_text) VALUES('delete', old.id, old.title, old.notes_text);
END;
CREATE TRIGGER IF NOT EXISTS todos_au AFTER UPDATE ON todos BEGIN
  INSERT INTO todos_fts(todos_fts, rowid, title, notes_text) VALUES('delete', old.id, old.title, old.notes_text);
  INSERT INTO todos_fts(rowid, title, notes_text) VALUES (new.id, new.title, new.notes_text);
END;
