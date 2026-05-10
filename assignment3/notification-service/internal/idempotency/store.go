package idempotency

import (
	"database/sql"
	"errors"
)

// Store persists processed event IDs to prevent duplicate handling.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// IsSeen returns true if the event has already been processed.
func (s *Store) IsSeen(eventID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1)`,
		eventID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// MarkSeen records an event ID as processed.
// Returns an error if insertion fails (e.g., duplicate — safe to ignore on unique constraint).
func (s *Store) MarkSeen(eventID string) error {
	_, err := s.db.Exec(
		`INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING`,
		eventID,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}
