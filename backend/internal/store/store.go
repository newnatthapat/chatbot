// Package store persists Conversations and Messages in SQLite. Only fully
// completed assistant Messages are ever appended (see the spec's streaming
// decision) — a failed or interrupted Provider call simply appends nothing.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a Conversation doesn't exist.
var ErrNotFound = errors.New("store: not found")

type Conversation struct {
	ID        int64
	CreatedAt time.Time
}

type Message struct {
	ID             int64
	ConversationID int64
	Role           string
	Content        string
	CreatedAt      time.Time
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	// A single connection avoids two SQLite gotchas: an in-memory database
	// being invisible across pooled connections, and PRAGMA foreign_keys
	// being a per-connection setting.
	db.SetMaxOpenConns(1)

	const schema = `
	CREATE TABLE IF NOT EXISTS conversations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: migrate: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: enable foreign keys: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) CreateConversation(ctx context.Context) (Conversation, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `INSERT INTO conversations (created_at) VALUES (?)`, now.Format(time.RFC3339Nano))
	if err != nil {
		return Conversation{}, fmt.Errorf("store: create conversation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Conversation{}, fmt.Errorf("store: create conversation: %w", err)
	}
	return Conversation{ID: id, CreatedAt: now}, nil
}

func (s *Store) ListConversations(ctx context.Context) ([]Conversation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, created_at FROM conversations ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list conversations: %w", err)
	}
	defer rows.Close()

	convs := []Conversation{}
	for rows.Next() {
		var c Conversation
		var createdAt string
		if err := rows.Scan(&c.ID, &createdAt); err != nil {
			return nil, fmt.Errorf("store: list conversations: %w", err)
		}
		c.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("store: list conversations: %w", err)
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

func (s *Store) GetConversation(ctx context.Context, id int64) (Conversation, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, created_at FROM conversations WHERE id = ?`, id)
	var c Conversation
	var createdAt string
	if err := row.Scan(&c.ID, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Conversation{}, ErrNotFound
		}
		return Conversation{}, fmt.Errorf("store: get conversation: %w", err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Conversation{}, fmt.Errorf("store: get conversation: %w", err)
	}
	c.CreatedAt = parsed
	return c, nil
}

func (s *Store) DeleteConversation(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM conversations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete conversation: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: delete conversation: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListMessages(ctx context.Context, conversationID int64) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, conversation_id, role, content, created_at FROM messages WHERE conversation_id = ? ORDER BY id ASC`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("store: list messages: %w", err)
	}
	defer rows.Close()

	msgs := []Message{}
	for rows.Next() {
		var m Message
		var createdAt string
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &createdAt); err != nil {
			return nil, fmt.Errorf("store: list messages: %w", err)
		}
		m.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("store: list messages: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *Store) AppendMessage(ctx context.Context, conversationID int64, role, content string) (Message, error) {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (conversation_id, role, content, created_at) VALUES (?, ?, ?, ?)`,
		conversationID, role, content, now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Message{}, fmt.Errorf("store: append message: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Message{}, fmt.Errorf("store: append message: %w", err)
	}
	return Message{ID: id, ConversationID: conversationID, Role: role, Content: content, CreatedAt: now}, nil
}
