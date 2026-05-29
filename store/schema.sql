-- todos table
-- Note: column name is unchanged (`todos`) but holds items of multiple kinds
-- (todo / note / scratch). Discriminator is `kind`. `notes_md` is preserved
-- as deadweight after the v7-v10 migration window for upgrade safety; it is
-- not read or written by current code and gets dropped in a follow-up release.
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
  notes_doc TEXT DEFAULT '',
  notes_html TEXT DEFAULT '',
  notes_html_version INTEGER NOT NULL DEFAULT 0,
  section TEXT DEFAULT 'anytime',
  pinned INTEGER NOT NULL DEFAULT 0,
  kind TEXT NOT NULL DEFAULT 'todo',
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

-- references between items
CREATE TABLE IF NOT EXISTS refs (
  source_id INTEGER NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
  target_id INTEGER NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
  PRIMARY KEY (source_id, target_id)
);

-- kinds registry. Plugin-registered kinds get plugin_id; core kinds have is_core=1.
CREATE TABLE IF NOT EXISTS kinds (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  icon TEXT NOT NULL DEFAULT '',
  default_view TEXT NOT NULL DEFAULT '',
  plugin_id TEXT DEFAULT NULL,
  is_core INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT OR IGNORE INTO kinds (name, display_name, icon, is_core) VALUES
  ('todo',    'Todo',    'check-square', 1),
  ('note',    'Note',    'file-text',    1),
  ('scratch', 'Scratch', 'edit',         1);

-- FTS5: standalone (no external content). Written by the application layer
-- via updateFTS / deleteFTS — no triggers. Source of `notes_text` is the
-- plain-text projection of notes_doc (or stripHTML(notes_md) during the
-- backfill transition).
CREATE VIRTUAL TABLE IF NOT EXISTS todos_fts USING fts5(title, notes_text);
