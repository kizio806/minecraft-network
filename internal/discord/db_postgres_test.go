package discord

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockResult struct {
	rowsAffected int64
}

func (m mockResult) LastInsertId() (int64, error) { return 0, nil }
func (m mockResult) RowsAffected() (int64, error) { return m.rowsAffected, nil }

type mockRows struct {
	columns []string
	data    [][]driver.Value
	pos     int
}

func (m *mockRows) Columns() []string { return m.columns }
func (m *mockRows) Close() error      { return nil }
func (m *mockRows) Next(dest []driver.Value) error {
	if m.pos >= len(m.data) {
		return io.EOF
	}
	copy(dest, m.data[m.pos])
	m.pos++
	return nil
}

type mockStmt struct {
	query string
	conn  *mockConn
}

func (s *mockStmt) Close() error  { return nil }
func (s *mockStmt) NumInput() int { return -1 }
func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.conn.exec(s.query, args)
}
func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.conn.query(s.query, args)
}

type mockConn struct {
	mu           sync.Mutex
	data         map[string]string
	reports      map[int64]*Report
	nextReportID int64
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return &mockStmt{query: query, conn: c}, nil
}
func (c *mockConn) Close() error              { return nil }
func (c *mockConn) Begin() (driver.Tx, error) { return nil, errors.New("tx not implemented") }

func (c *mockConn) exec(_ string, args []driver.Value) (driver.Result, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, arg := range args {
		if s, ok := arg.(string); ok && strings.Contains(s, "0_rows_affected") {
			return mockResult{rowsAffected: 0}, nil
		}
	}
	return mockResult{rowsAffected: 1}, nil
}

func (c *mockConn) query(query string, args []driver.Value) (driver.Rows, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch {
	case strings.Contains(query, "SELECT username FROM discord_links"):
		if len(args) > 0 && args[0] == "discord_existing" {
			return &mockRows{
				columns: []string{"username"},
				data:    [][]driver.Value{{"Steve"}},
			}, nil
		}
		if len(args) > 0 && args[0] == "discord_error" {
			return nil, errors.New("db error")
		}
		return &mockRows{columns: []string{"username"}, data: nil}, nil

	case strings.Contains(query, "INSERT INTO discord_reports"):
		c.nextReportID++
		now := time.Now()
		return &mockRows{
			columns: []string{"id", "created_at"},
			data:    [][]driver.Value{{c.nextReportID, now}},
		}, nil

	case strings.Contains(query, "SELECT id, reporter_id"):
		if len(args) > 0 && args[0] == int64(42) {
			now := time.Now()
			return &mockRows{
				columns: []string{"id", "reporter_id", "reporter_tag", "target_player", "reason", "status", "resolved_by", "created_at", "resolved_at"},
				data: [][]driver.Value{
					{int64(42), "user_1", "User#1", "Cheater", "Speedhack", "pending", "", now, nil},
				},
			}, nil
		}
		if len(args) > 0 && args[0] == int64(999) {
			return nil, errors.New("db query error")
		}
		return &mockRows{columns: []string{"id", "reporter_id", "reporter_tag", "target_player", "reason", "status", "resolved_by", "created_at", "resolved_at"}, data: nil}, nil

	case strings.Contains(query, "SELECT primary_group FROM luckperms_players"):
		if len(args) > 0 && args[0] == "kizio" {
			return &mockRows{
				columns: []string{"primary_group"},
				data:    [][]driver.Value{{"owner"}},
			}, nil
		}
		if len(args) > 0 && args[0] == "emptygroup" {
			return &mockRows{
				columns: []string{"primary_group"},
				data:    [][]driver.Value{{""}},
			}, nil
		}
		if len(args) > 0 && args[0] == "db_error_player" {
			return nil, errors.New("luckperms db error")
		}
		return &mockRows{columns: []string{"primary_group"}, data: nil}, nil
	}

	return &mockRows{columns: []string{"dummy"}, data: nil}, nil
}

type mockDriver struct{}

func (d *mockDriver) Open(_ string) (driver.Conn, error) {
	return &mockConn{
		data:         make(map[string]string),
		reports:      make(map[int64]*Report),
		nextReportID: 0,
	}, nil
}

func init() {
	sql.Register("test_pg_mock", &mockDriver{})
}

func TestPostgresStore_MockOperations(t *testing.T) {
	db, err := sql.Open("test_pg_mock", "")
	if err != nil {
		t.Fatalf("failed to open mock db: %v", err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	ctx := context.Background()

	t.Run("create tables", func(t *testing.T) {
		if err := store.CreateTables(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("link account success", func(t *testing.T) {
		if err := store.LinkAccount(ctx, "discord_new", "Player1"); err != nil {
			t.Fatalf("unexpected link error: %v", err)
		}
	})

	t.Run("link account conflict", func(t *testing.T) {
		err := store.LinkAccount(ctx, "discord_existing", "Alex")
		if err == nil {
			t.Fatalf("expected conflict error, got nil")
		}
	})

	t.Run("link account db error", func(t *testing.T) {
		err := store.LinkAccount(ctx, "discord_error", "Steve")
		if err == nil {
			t.Fatalf("expected db error, got nil")
		}
	})

	t.Run("get linked account found", func(t *testing.T) {
		nick, err := store.GetLinkedAccount(ctx, "discord_existing")
		if err != nil || nick != "Steve" {
			t.Fatalf("expected Steve, got %s (err: %v)", nick, err)
		}
	})

	t.Run("get linked account not found", func(t *testing.T) {
		_, err := store.GetLinkedAccount(ctx, "discord_missing")
		if err == nil {
			t.Fatalf("expected error for missing account, got nil")
		}
	})

	t.Run("get linked account query error", func(t *testing.T) {
		_, err := store.GetLinkedAccount(ctx, "discord_error")
		if err == nil {
			t.Fatalf("expected error for db error, got nil")
		}
	})

	t.Run("add and remove whitelist", func(t *testing.T) {
		if err := store.AddToWhitelist(ctx, "Player1"); err != nil {
			t.Fatalf("add whitelist error: %v", err)
		}
		if err := store.RemoveFromWhitelist(ctx, "Player1"); err != nil {
			t.Fatalf("remove whitelist error: %v", err)
		}
	})

	t.Run("save and get report", func(t *testing.T) {
		rep := &Report{
			ReporterID:   "user_1",
			ReporterTag:  "User#1",
			TargetPlayer: "Cheater",
			Reason:       "Speedhack",
		}
		if err := store.SaveReport(ctx, rep); err != nil {
			t.Fatalf("save report failed: %v", err)
		}
		if rep.ID != 1 {
			t.Fatalf("expected report ID 1, got %d", rep.ID)
		}

		fetched, err := store.GetReport(ctx, 42)
		if err != nil {
			t.Fatalf("get report failed: %v", err)
		}
		if fetched.TargetPlayer != "Cheater" {
			t.Fatalf("unexpected target player: %s", fetched.TargetPlayer)
		}
	})

	t.Run("get report not found", func(t *testing.T) {
		_, err := store.GetReport(ctx, 404)
		if err == nil {
			t.Fatalf("expected error for missing report, got nil")
		}
	})

	t.Run("get report query error", func(t *testing.T) {
		_, err := store.GetReport(ctx, 999)
		if err == nil {
			t.Fatalf("expected query error, got nil")
		}
	})

	t.Run("update report status", func(t *testing.T) {
		if err := store.UpdateReportStatus(ctx, 1, "resolved", "Admin"); err != nil {
			t.Fatalf("update status failed: %v", err)
		}
	})

	t.Run("update report status zero rows", func(t *testing.T) {
		err := store.UpdateReportStatus(ctx, 1, "0_rows_affected", "Admin")
		if err == nil {
			t.Fatalf("expected error for zero rows affected, got nil")
		}
	})

	t.Run("get player rank found", func(t *testing.T) {
		rank, err := store.GetPlayerRank(ctx, "kizio")
		if err != nil || rank != "owner" {
			t.Fatalf("expected owner, got %s (err: %v)", rank, err)
		}
	})

	t.Run("get player rank empty fallback", func(t *testing.T) {
		rank, err := store.GetPlayerRank(ctx, "emptygroup")
		if err != nil || rank != "default" {
			t.Fatalf("expected default for empty group, got %s (err: %v)", rank, err)
		}
	})

	t.Run("get player rank not found default", func(t *testing.T) {
		rank, err := store.GetPlayerRank(ctx, "unknown_player")
		if err != nil || rank != "default" {
			t.Fatalf("expected default for unknown player, got %s (err: %v)", rank, err)
		}
	})

	t.Run("get player rank db error", func(t *testing.T) {
		rank, err := store.GetPlayerRank(ctx, "db_error_player")
		if err == nil || rank != "default" {
			t.Fatalf("expected error for db failure, got rank=%s err=%v", rank, err)
		}
	})

	t.Run("close", func(t *testing.T) {
		if err := store.Close(); err != nil {
			t.Fatalf("close failed: %v", err)
		}
	})
}

func TestNewPostgresStore_ErrorHandling(t *testing.T) {
	_, err := NewPostgresStore("postgres://bad_user:bad_pass@127.0.0.1:54329/nonexistent?sslmode=disable&connect_timeout=1")
	if err == nil {
		t.Fatalf("expected error connecting to non-existent postgres, got nil")
	}
}
