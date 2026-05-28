// Copyright 2019 FairwindsOps Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package history persists periodic snapshots of every container's current
// vs recommended resources, so the dashboard can render trends over time.
package history

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // register the pure-Go sqlite driver
)

// Snapshot is a single row written by the collector. Resource quantity fields
// are pointers so the schema can distinguish "not set" from zero.
type Snapshot struct {
	Timestamp    time.Time
	Namespace    string
	WorkloadKind string
	WorkloadName string
	Container    string

	CPURequestM *int64 // millicores
	CPULimitM   *int64
	MemRequestB *int64 // bytes
	MemLimitB   *int64

	CPUTargetM *int64
	MemTargetB *int64
	CPULowerM  *int64
	CPUUpperM  *int64
	MemLowerB  *int64
	MemUpperB  *int64
}

// Point is a single time-series datum returned by Series.
type Point struct {
	TS         int64 `json:"ts"`
	CPURequest int64 `json:"cpuRequest"`
	CPULimit   int64 `json:"cpuLimit"`
	MemRequest int64 `json:"memRequest"`
	MemLimit   int64 `json:"memLimit"`
	CPUTarget  int64 `json:"cpuTarget"`
	MemTarget  int64 `json:"memTarget"`
	CPULower   int64 `json:"cpuLower"`
	CPUUpper   int64 `json:"cpuUpper"`
	MemLower   int64 `json:"memLower"`
	MemUpper   int64 `json:"memUpper"`
}

// Store is the SQLite-backed history persistence layer. Use Open to create
// one; safe for concurrent use by multiple goroutines.
type Store struct {
	db   *sql.DB
	path string
}

// Open returns a Store backed by a SQLite database at the given path. The
// file is created if missing along with any parent directories.
// WAL mode is enabled so multiple readers can run while the collector writes.
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("history: path must not be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("history: ensure parent dir: %w", err)
	}
	// _journal_mode=WAL allows concurrent reads while one writer is active.
	// _busy_timeout=5000 retries briefly on lock contention so the collector
	// and read endpoint don't trip over each other.
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("history: open sqlite: %w", err)
	}
	// One writer connection prevents "database is locked" surprises under
	// the modernc driver (which serializes writes anyway).
	db.SetMaxOpenConns(8)

	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("history: ping sqlite: %w", err)
	}

	s := &Store{db: db, path: path}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Path returns the on-disk path of the underlying database file.
func (s *Store) Path() string { return s.path }

// Close releases the underlying connection pool.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS snapshots (
			ts INTEGER NOT NULL,
			namespace TEXT NOT NULL,
			workload_kind TEXT NOT NULL,
			workload_name TEXT NOT NULL,
			container TEXT NOT NULL,
			cpu_request_m INTEGER,
			cpu_limit_m INTEGER,
			mem_request_b INTEGER,
			mem_limit_b INTEGER,
			cpu_target_m INTEGER,
			mem_target_b INTEGER,
			cpu_lower_m INTEGER,
			cpu_upper_m INTEGER,
			mem_lower_b INTEGER,
			mem_upper_b INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ns_wl_c_ts
			ON snapshots(namespace, workload_name, container, ts)`,
		`CREATE INDEX IF NOT EXISTS idx_ts
			ON snapshots(ts)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("history: migrate: %w", err)
		}
	}
	return nil
}

// Write inserts a batch of snapshots in a single transaction. Empty batches
// are a no-op.
func (s *Store) Write(ctx context.Context, snapshots []Snapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("history: begin tx: %w", err)
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO snapshots(
		ts, namespace, workload_kind, workload_name, container,
		cpu_request_m, cpu_limit_m, mem_request_b, mem_limit_b,
		cpu_target_m, mem_target_b,
		cpu_lower_m, cpu_upper_m, mem_lower_b, mem_upper_b
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("history: prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, sn := range snapshots {
		if _, err := stmt.ExecContext(ctx,
			sn.Timestamp.Unix(),
			sn.Namespace, sn.WorkloadKind, sn.WorkloadName, sn.Container,
			nullableInt(sn.CPURequestM), nullableInt(sn.CPULimitM),
			nullableInt(sn.MemRequestB), nullableInt(sn.MemLimitB),
			nullableInt(sn.CPUTargetM), nullableInt(sn.MemTargetB),
			nullableInt(sn.CPULowerM), nullableInt(sn.CPUUpperM),
			nullableInt(sn.MemLowerB), nullableInt(sn.MemUpperB),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("history: exec insert: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: commit: %w", err)
	}
	return nil
}

// Series returns all points for one (namespace, workload, container) tuple
// since the given time, ordered oldest-first.
func (s *Store) Series(ctx context.Context, namespace, workloadKind, workloadName, container string, since time.Time) ([]Point, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
		ts,
		IFNULL(cpu_request_m, 0), IFNULL(cpu_limit_m, 0),
		IFNULL(mem_request_b, 0), IFNULL(mem_limit_b, 0),
		IFNULL(cpu_target_m, 0), IFNULL(mem_target_b, 0),
		IFNULL(cpu_lower_m, 0), IFNULL(cpu_upper_m, 0),
		IFNULL(mem_lower_b, 0), IFNULL(mem_upper_b, 0)
		FROM snapshots
		WHERE namespace = ? AND workload_kind = ? AND workload_name = ? AND container = ? AND ts >= ?
		ORDER BY ts ASC`,
		namespace, workloadKind, workloadName, container, since.Unix())
	if err != nil {
		return nil, fmt.Errorf("history: query: %w", err)
	}
	defer rows.Close()

	var out []Point
	for rows.Next() {
		var p Point
		if err := rows.Scan(
			&p.TS,
			&p.CPURequest, &p.CPULimit,
			&p.MemRequest, &p.MemLimit,
			&p.CPUTarget, &p.MemTarget,
			&p.CPULower, &p.CPUUpper,
			&p.MemLower, &p.MemUpper,
		); err != nil {
			return nil, fmt.Errorf("history: scan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history: rows: %w", err)
	}
	return out, nil
}

// Prune deletes snapshots older than the given cutoff and returns the number
// of rows removed.
func (s *Store) Prune(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM snapshots WHERE ts < ?`, cutoff.Unix())
	if err != nil {
		return 0, fmt.Errorf("history: prune: %w", err)
	}
	return res.RowsAffected()
}

func nullableInt(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
