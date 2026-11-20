package discord

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// Report represents a player-submitted moderation report.
type Report struct {
	ID           int64      `json:"id"`
	ReporterID   string     `json:"reporter_id"`
	ReporterTag  string     `json:"reporter_tag"`
	TargetPlayer string     `json:"target_player"`
	Reason       string     `json:"reason"`
	Status       string     `json:"status"` // "pending", "resolved", "dismissed"
	ResolvedBy   string     `json:"resolved_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

// Store defines persistence operations for Discord account links, whitelists, reports, and rank lookups.
type Store interface {
	AccountLinker
	Close() error
	CreateTables(ctx context.Context) error
	SaveReport(ctx context.Context, r *Report) error
	GetReport(ctx context.Context, id int64) (*Report, error)
	UpdateReportStatus(ctx context.Context, id int64, status, resolvedBy string) error
	GetPlayerRank(ctx context.Context, username string) (string, error)
}

// PostgresStore implements Store backed by PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore connects to PostgreSQL and verifies the schema.
func NewPostgresStore(dbURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	store := &PostgresStore{db: db}
	if err := store.CreateTables(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return store, nil
}

// CreateTables provisions the necessary Discord tables if they do not exist.
func (s *PostgresStore) CreateTables(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS discord_links (
		discord_id VARCHAR(64) PRIMARY KEY,
		username VARCHAR(64) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_discord_links_user ON discord_links (LOWER(username));

	CREATE TABLE IF NOT EXISTS discord_whitelist (
		username VARCHAR(64) PRIMARY KEY,
		added_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS discord_reports (
		id BIGSERIAL PRIMARY KEY,
		reporter_id VARCHAR(64) NOT NULL,
		reporter_tag VARCHAR(100) NOT NULL,
		target_player VARCHAR(64) NOT NULL,
		reason TEXT NOT NULL,
		status VARCHAR(32) NOT NULL DEFAULT 'pending',
		resolved_by VARCHAR(64),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		resolved_at TIMESTAMPTZ
	);
	`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// LinkAccount stores the Discord-to-Minecraft link, rejecting conflicts.
func (s *PostgresStore) LinkAccount(ctx context.Context, discordID, username string) error {
	var existing string
	err := s.db.QueryRowContext(ctx, "SELECT username FROM discord_links WHERE discord_id = $1", discordID).Scan(&existing)
	if err == nil && !strings.EqualFold(existing, username) {
		return fmt.Errorf("already linked to %s", existing)
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check existing link: %w", err)
	}

	query := `
	INSERT INTO discord_links (discord_id, username, created_at)
	VALUES ($1, $2, NOW())
	ON CONFLICT (discord_id) DO UPDATE SET username = EXCLUDED.username;
	`
	_, err = s.db.ExecContext(ctx, query, discordID, username)
	return err
}

// GetLinkedAccount retrieves the Minecraft username linked to the given Discord ID.
func (s *PostgresStore) GetLinkedAccount(ctx context.Context, discordID string) (string, error) {
	var username string
	err := s.db.QueryRowContext(ctx, "SELECT username FROM discord_links WHERE discord_id = $1", discordID).Scan(&username)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("account not linked")
	}
	if err != nil {
		return "", fmt.Errorf("query linked account: %w", err)
	}
	return username, nil
}

// AddToWhitelist registers a player nickname on the whitelist.
func (s *PostgresStore) AddToWhitelist(ctx context.Context, username string) error {
	query := `INSERT INTO discord_whitelist (username, added_at) VALUES ($1, NOW()) ON CONFLICT (username) DO NOTHING;`
	_, err := s.db.ExecContext(ctx, query, username)
	return err
}

// RemoveFromWhitelist removes a player nickname from the whitelist.
func (s *PostgresStore) RemoveFromWhitelist(ctx context.Context, username string) error {
	query := `DELETE FROM discord_whitelist WHERE LOWER(username) = LOWER($1);`
	_, err := s.db.ExecContext(ctx, query, username)
	return err
}

// SaveReport records a moderation report and populates its generated ID.
func (s *PostgresStore) SaveReport(ctx context.Context, r *Report) error {
	query := `
	INSERT INTO discord_reports (reporter_id, reporter_tag, target_player, reason, status, created_at)
	VALUES ($1, $2, $3, $4, 'pending', NOW())
	RETURNING id, created_at;
	`
	return s.db.QueryRowContext(ctx, query, r.ReporterID, r.ReporterTag, r.TargetPlayer, r.Reason).Scan(&r.ID, &r.CreatedAt)
}

// GetReport fetches a moderation report by ID.
func (s *PostgresStore) GetReport(ctx context.Context, id int64) (*Report, error) {
	query := `
	SELECT id, reporter_id, reporter_tag, target_player, reason, status, COALESCE(resolved_by, ''), created_at, resolved_at
	FROM discord_reports WHERE id = $1;
	`
	var r Report
	var resBy string
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID, &r.ReporterID, &r.ReporterTag, &r.TargetPlayer, &r.Reason, &r.Status, &resBy, &r.CreatedAt, &r.ResolvedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("report #%d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query report: %w", err)
	}
	r.ResolvedBy = resBy
	return &r, nil
}

// UpdateReportStatus sets the status and resolution timestamp for a report.
func (s *PostgresStore) UpdateReportStatus(ctx context.Context, id int64, status, resolvedBy string) error {
	query := `
	UPDATE discord_reports
	SET status = $2, resolved_by = $3, resolved_at = NOW()
	WHERE id = $1;
	`
	res, err := s.db.ExecContext(ctx, query, id, status, resolvedBy)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return fmt.Errorf("report #%d not found", id)
	}
	return nil
}

// GetPlayerRank inspects luckperms_players to find the primary group of a user.
func (s *PostgresStore) GetPlayerRank(ctx context.Context, username string) (string, error) {
	var group string
	query := `SELECT primary_group FROM luckperms_players WHERE LOWER(username) = LOWER($1) LIMIT 1;`
	err := s.db.QueryRowContext(ctx, query, username).Scan(&group)
	if errors.Is(err, sql.ErrNoRows) {
		return "default", nil
	}
	if err != nil {
		return "default", fmt.Errorf("query luckperms rank: %w", err)
	}
	if group == "" {
		return "default", nil
	}
	return group, nil
}

// Close closes the underlying database pool.
func (s *PostgresStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// MemoryStore provides an in-memory implementation of Store for testing and standby.
type MemoryStore struct {
	mu           sync.RWMutex
	linked       map[string]string
	whitelist    map[string]bool
	reports      map[int64]*Report
	ranks        map[string]string
	nextReportID int64
}

// NewMemoryStore initializes a thread-safe MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		linked:       make(map[string]string),
		whitelist:    make(map[string]bool),
		reports:      make(map[int64]*Report),
		ranks:        make(map[string]string),
		nextReportID: 1,
	}
}

func (m *MemoryStore) Close() error {
	return nil
}

func (m *MemoryStore) CreateTables(_ context.Context) error {
	return nil
}

func (m *MemoryStore) LinkAccount(_ context.Context, discordID, username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.linked[discordID]; ok && !strings.EqualFold(existing, username) {
		return fmt.Errorf("already linked to %s", existing)
	}
	m.linked[discordID] = username
	return nil
}

func (m *MemoryStore) GetLinkedAccount(_ context.Context, discordID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if nick, ok := m.linked[discordID]; ok {
		return nick, nil
	}
	return "", fmt.Errorf("account not linked")
}

func (m *MemoryStore) AddToWhitelist(_ context.Context, username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.whitelist[strings.ToLower(username)] = true
	return nil
}

func (m *MemoryStore) RemoveFromWhitelist(_ context.Context, username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.whitelist, strings.ToLower(username))
	return nil
}

func (m *MemoryStore) SaveReport(_ context.Context, r *Report) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r.ID = m.nextReportID
	m.nextReportID++
	r.Status = "pending"
	r.CreatedAt = time.Now()
	clone := *r
	m.reports[r.ID] = &clone
	return nil
}

func (m *MemoryStore) GetReport(_ context.Context, id int64) (*Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if rep, ok := m.reports[id]; ok {
		clone := *rep
		return &clone, nil
	}
	return nil, fmt.Errorf("report #%d not found", id)
}

func (m *MemoryStore) UpdateReportStatus(_ context.Context, id int64, status, resolvedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rep, ok := m.reports[id]
	if !ok {
		return fmt.Errorf("report #%d not found", id)
	}
	now := time.Now()
	rep.Status = status
	rep.ResolvedBy = resolvedBy
	rep.ResolvedAt = &now
	return nil
}

func (m *MemoryStore) GetPlayerRank(_ context.Context, username string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.ranks[strings.ToLower(username)]; ok && r != "" {
		return r, nil
	}
	return "default", nil
}

// SetPlayerRank stores a mock LuckPerms rank for testing purposes.
func (m *MemoryStore) SetPlayerRank(username, rank string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ranks[strings.ToLower(username)] = rank
}
