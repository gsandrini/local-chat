package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Session represents a saved chat session
type Session struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	Model          string `json:"model"`
	CreatedAt      string `json:"created_at"`
	SystemFileName string `json:"system_file_name"`
}

// initDB opens (or creates) the SQLite database and runs migrations
func initDB() (*sql.DB, error) {
	dir, err := appDataDir()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve data dir: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create data dir: %w", err)
	}

	dbPath := filepath.Join(dir, "history.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("cannot open db: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

// appDataDir returns ~/.local/share/local-chat on Linux,
// ~/Library/Application Support/local-chat on macOS,
// %APPDATA%\local-chat on Windows.
func appDataDir() (string, error) {
	// os.UserConfigDir returns the right base on every platform
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "local-chat"), nil
}

// migrate creates the tables if they do not exist yet and applies incremental migrations
func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			title            TEXT    NOT NULL,
			model            TEXT    NOT NULL DEFAULT '',
			created_at       TEXT    NOT NULL
		);

		CREATE TABLE IF NOT EXISTS messages (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			role       TEXT    NOT NULL,
			content    TEXT    NOT NULL,
			created_at TEXT    NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
	`)
	if err != nil {
		return err
	}

	// Incremental migration: add system_prompt and system_file_name columns if missing
	for _, col := range []struct{ name, def string }{
		{"system_prompt", "TEXT NOT NULL DEFAULT ''"},
		{"system_file_name", "TEXT NOT NULL DEFAULT ''"},
	} {
		var dummy string
		err := db.QueryRow(`SELECT ` + col.name + ` FROM sessions LIMIT 1`).Scan(&dummy)
		if err != nil && strings.Contains(err.Error(), "no such column") {
			if _, err2 := db.Exec(`ALTER TABLE sessions ADD COLUMN ` + col.name + ` ` + col.def); err2 != nil {
				return err2
			}
		}
	}

	return nil
}

// --- Methods exposed to Wails / frontend ---

// GetSessions returns all sessions ordered by most recent first
func (a *App) GetSessions() ([]Session, error) {
	rows, err := a.db.Query(
		`SELECT id, title, model, created_at, system_file_name FROM sessions ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.Title, &s.Model, &s.CreatedAt, &s.SystemFileName); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// LoadSession loads all messages of a session into the in-memory history
// and returns them to the frontend; also restores the system prompt
func (a *App) LoadSession(sessionID int64) ([]Message, error) {
	// Restore system prompt and file name for this session
	var systemPrompt, systemFileName string
	err := a.db.QueryRow(
		`SELECT system_prompt, system_file_name FROM sessions WHERE id = ?`, sessionID,
	).Scan(&systemPrompt, &systemFileName)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	a.systemPrompt = systemPrompt
	a.systemFileName = systemFileName

	rows, err := a.db.Query(
		`SELECT role, content FROM messages WHERE session_id = ? ORDER BY id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	// Replace the in-memory history so the next Chat() call has full context
	a.history = messages
	a.currentSessionID = sessionID

	return messages, nil
}

// DeleteSession removes a session and all its messages (CASCADE)
func (a *App) DeleteSession(sessionID int64) error {
	_, err := a.db.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
	if err != nil {
		return err
	}
	// If the deleted session was the active one, reset state
	if a.currentSessionID == sessionID {
		a.currentSessionID = 0
		a.history = []Message{}
		a.systemPrompt = ""
		a.systemFileName = ""
	}
	return nil
}

// NewSession resets the in-memory history and clears the current session ID
// so the next Chat() call will create a fresh session in the DB
func (a *App) NewSession() {
	a.history = []Message{}
	a.currentSessionID = 0
	a.systemPrompt = ""
	a.systemFileName = ""
}

// RenameSession updates the title of a session
func (a *App) RenameSession(sessionID int64, title string) error {
	_, err := a.db.Exec(`UPDATE sessions SET title = ? WHERE id = ?`, title, sessionID)
	return err
}

// --- Internal helpers ---

// saveSession persists the current in-memory history to SQLite.
// Called automatically at the end of every successful Chat().
func (a *App) saveSession(model string) error {
	now := time.Now().Format(time.RFC3339)

	if a.currentSessionID == 0 {
		// Derive a title from the first user message (max 60 chars)
		title := "Chat"
		for _, m := range a.history {
			if m.Role == "user" {
				title = truncate(m.Content, 60)
				break
			}
		}

		res, err := a.db.Exec(
			`INSERT INTO sessions (title, model, created_at, system_prompt, system_file_name) VALUES (?, ?, ?, ?, ?)`,
			title, model, now, a.systemPrompt, a.systemFileName,
		)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		a.currentSessionID = id
	} else {
		// Update system prompt/file name in case they changed during the session
		if _, err := a.db.Exec(
			`UPDATE sessions SET system_prompt = ?, system_file_name = ? WHERE id = ?`,
			a.systemPrompt, a.systemFileName, a.currentSessionID,
		); err != nil {
			return err
		}
	}

	// Delete existing messages for this session and rewrite them.
	if _, err := a.db.Exec(
		`DELETE FROM messages WHERE session_id = ?`, a.currentSessionID,
	); err != nil {
		return err
	}

	stmt, err := a.db.Prepare(
		`INSERT INTO messages (session_id, role, content, created_at) VALUES (?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range a.history {
		if _, err := stmt.Exec(a.currentSessionID, m.Role, m.Content, now); err != nil {
			return err
		}
	}

	return nil
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
