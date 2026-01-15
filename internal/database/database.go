package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/emirozbir/micro-sre/internal/models"
)

const schema = `
CREATE TABLE IF NOT EXISTS analyses (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at DATETIME NOT NULL,
	alert_name TEXT NOT NULL,
	namespace TEXT NOT NULL,
	pod_name TEXT NOT NULL,
	severity TEXT NOT NULL,
	alert_started_at DATETIME NOT NULL,
	root_cause TEXT NOT NULL,
	confidence TEXT NOT NULL,
	analysis_json TEXT NOT NULL,
	UNIQUE(namespace, pod_name, alert_started_at)
);

CREATE INDEX IF NOT EXISTS idx_created_at ON analyses(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_namespace_pod ON analyses(namespace, pod_name);
CREATE INDEX IF NOT EXISTS idx_severity ON analyses(severity);
`

type DB struct {
	conn *sql.DB
}

type StoredAnalysis struct {
	ID              int64
	CreatedAt       time.Time
	AlertName       string
	Namespace       string
	PodName         string
	Severity        string
	AlertStartedAt  time.Time
	RootCause       string
	Confidence      string
	AnalysisResult  models.AnalysisResult
}

// New creates a new database connection and initializes the schema
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys and WAL mode for better performance
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if _, err := conn.Exec("PRAGMA journal_mode = WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Create schema
	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// SaveAnalysis saves an analysis result to the database
func (db *DB) SaveAnalysis(result *models.AnalysisResult) (int64, error) {
	analysisJSON, err := json.Marshal(result)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal analysis: %w", err)
	}

	query := `
		INSERT INTO analyses (
			created_at, alert_name, namespace, pod_name, severity,
			alert_started_at, root_cause, confidence, analysis_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(namespace, pod_name, alert_started_at)
		DO UPDATE SET
			created_at = excluded.created_at,
			alert_name = excluded.alert_name,
			severity = excluded.severity,
			root_cause = excluded.root_cause,
			confidence = excluded.confidence,
			analysis_json = excluded.analysis_json
	`

	res, err := db.conn.Exec(
		query,
		time.Now(),
		result.Alert.Name,
		result.Alert.Namespace,
		result.Alert.Pod,
		result.Alert.Severity,
		result.Alert.StartedAt,
		result.Analysis.RootCause,
		result.Analysis.Confidence,
		string(analysisJSON),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert analysis: %w", err)
	}

	return res.LastInsertId()
}

// GetAnalysis retrieves a single analysis by ID
func (db *DB) GetAnalysis(id int64) (*StoredAnalysis, error) {
	query := `
		SELECT id, created_at, alert_name, namespace, pod_name, severity,
		       alert_started_at, root_cause, confidence, analysis_json
		FROM analyses
		WHERE id = ?
	`

	var stored StoredAnalysis
	var analysisJSON string

	err := db.conn.QueryRow(query, id).Scan(
		&stored.ID,
		&stored.CreatedAt,
		&stored.AlertName,
		&stored.Namespace,
		&stored.PodName,
		&stored.Severity,
		&stored.AlertStartedAt,
		&stored.RootCause,
		&stored.Confidence,
		&analysisJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query analysis: %w", err)
	}

	if err := json.Unmarshal([]byte(analysisJSON), &stored.AnalysisResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal analysis: %w", err)
	}

	return &stored, nil
}

// ListAnalyses retrieves all analyses with pagination
func (db *DB) ListAnalyses(limit, offset int) ([]StoredAnalysis, error) {
	query := `
		SELECT id, created_at, alert_name, namespace, pod_name, severity,
		       alert_started_at, root_cause, confidence, analysis_json
		FROM analyses
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.conn.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query analyses: %w", err)
	}
	defer rows.Close()

	var analyses []StoredAnalysis
	for rows.Next() {
		var stored StoredAnalysis
		var analysisJSON string

		err := rows.Scan(
			&stored.ID,
			&stored.CreatedAt,
			&stored.AlertName,
			&stored.Namespace,
			&stored.PodName,
			&stored.Severity,
			&stored.AlertStartedAt,
			&stored.RootCause,
			&stored.Confidence,
			&analysisJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if err := json.Unmarshal([]byte(analysisJSON), &stored.AnalysisResult); err != nil {
			return nil, fmt.Errorf("failed to unmarshal analysis: %w", err)
		}

		analyses = append(analyses, stored)
	}

	return analyses, rows.Err()
}

// CountAnalyses returns the total number of analyses
func (db *DB) CountAnalyses() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM analyses").Scan(&count)
	return count, err
}

// DeleteAnalysis deletes an analysis by ID
func (db *DB) DeleteAnalysis(id int64) error {
	_, err := db.conn.Exec("DELETE FROM analyses WHERE id = ?", id)
	return err
}
