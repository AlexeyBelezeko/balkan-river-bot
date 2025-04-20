// Package repository provides data access implementations
package repository

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/abelzeko/water-bot/internal/entities"
	_ "github.com/mattn/go-sqlite3"
)

// RiverRepository defines the interface for river data persistence operations
type RiverRepository interface {
	SaveRiverData(data []entities.RiverData) error
	GetRiverDataByName(riverName string) ([]entities.RiverData, error)
	GetUniqueRivers() ([]string, error)
	GetLastUpdateTime() (time.Time, error)
	Close() error
}

// SQLiteRiverRepository implements RiverRepository using SQLite
type SQLiteRiverRepository struct {
	db     *sql.DB
	DBPath string
}

// NewSQLiteRiverRepository creates and initializes a new SQLite repository
func NewSQLiteRiverRepository(dbPath string) (*SQLiteRiverRepository, error) {
	if dbPath == "" {
		// Set default path if not specified
		dbDir := "data"
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %v", err)
		}
		dbPath = filepath.Join(dbDir, "riverdata.db")
	}

	log.Printf("Opening database at %s", dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Create river_data table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS river_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		river TEXT NOT NULL,
		station TEXT NOT NULL,
		water_level TEXT,
		water_temp TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(river, station, timestamp)
	);
	CREATE INDEX IF NOT EXISTS idx_river ON river_data(river);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON river_data(timestamp);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return &SQLiteRiverRepository{
		db:     db,
		DBPath: dbPath,
	}, nil
}

// Close closes the database connection
func (r *SQLiteRiverRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// SaveRiverData stores river data in the database
func (r *SQLiteRiverRepository) SaveRiverData(data []entities.RiverData) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Prepare SQL statement for inserting data
	stmt, err := tx.Prepare(`
		INSERT INTO river_data(river, station, water_level, water_temp, timestamp)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(river, station, timestamp) DO UPDATE SET
		water_level=excluded.water_level,
		water_temp=excluded.water_temp
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	// Insert each river data record
	for _, rd := range data {
		_, err := stmt.Exec(
			rd.River,
			rd.Station,
			rd.WaterLevel,
			rd.WaterTemp,
			rd.Timestamp,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert data for %s at %s: %v", rd.River, rd.Station, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("Successfully saved %d river data records", len(data))
	return nil
}

// GetRiverDataByName retrieves data for a specific river
func (r *SQLiteRiverRepository) GetRiverDataByName(riverName string) ([]entities.RiverData, error) {
	// Using subquery to get only the most recent data for each station
	query := `
		SELECT id, river, station, water_level, water_temp, timestamp
		FROM river_data
		WHERE river = ? AND (river, station, timestamp) IN (
			SELECT river, station, MAX(timestamp) 
			FROM river_data
			WHERE river = ?
			GROUP BY river, station
		)
		ORDER BY station`

	rows, err := r.db.Query(query, riverName, riverName)
	if err != nil {
		return nil, fmt.Errorf("failed to query river data for %s: %v", riverName, err)
	}
	defer rows.Close()

	var result []entities.RiverData
	for rows.Next() {
		var rd entities.RiverData
		if err := rows.Scan(
			&rd.ID,
			&rd.River,
			&rd.Station,
			&rd.WaterLevel,
			&rd.WaterTemp,
			&rd.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}
		result = append(result, rd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %v", err)
	}

	return result, nil
}

// GetUniqueRivers returns a list of all unique river names in the database
func (r *SQLiteRiverRepository) GetUniqueRivers() ([]string, error) {
	// Subquery to get only the most recent river data
	query := `
		SELECT DISTINCT river
		FROM river_data 
		WHERE (river, station, timestamp) IN (
			SELECT river, station, MAX(timestamp) 
			FROM river_data 
			GROUP BY river, station
		)
		ORDER BY river`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unique rivers: %v", err)
	}
	defer rows.Close()

	var rivers []string
	for rows.Next() {
		var river string
		if err := rows.Scan(&river); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}
		rivers = append(rivers, river)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %v", err)
	}

	return rivers, nil
}

// GetLastUpdateTime returns the most recent timestamp in the database
func (r *SQLiteRiverRepository) GetLastUpdateTime() (time.Time, error) {
	var timestampStr sql.NullString
	err := r.db.QueryRow("SELECT MAX(timestamp) FROM river_data").Scan(&timestampStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, nil // Return zero time if no data
		}
		return time.Time{}, fmt.Errorf("failed to get last update time: %v", err)
	}

	// If the timestamp is null/empty, return zero time
	if !timestampStr.Valid || timestampStr.String == "" {
		return time.Time{}, nil
	}

	// Try to parse the timestamp with different formats to handle potential timezone info
	var timestamp time.Time
	var parseErr error

	// First try with timezone (RFC3339 format)
	timestamp, parseErr = time.Parse(time.RFC3339, timestampStr.String)
	if parseErr == nil {
		return timestamp, nil
	}

	// Try SQLite DATETIME format without timezone
	timestamp, parseErr = time.ParseInLocation("2006-01-02 15:04:05", timestampStr.String, time.Local)
	if parseErr == nil {
		return timestamp, nil
	}

	// Try custom format with timezone suffix
	timestamp, parseErr = time.Parse("2006-01-02 15:04:05-07:00", timestampStr.String)
	if parseErr == nil {
		return timestamp, nil
	}

	// Try one more format with timezone suffix
	timestamp, parseErr = time.Parse("2006-01-02 15:04:05Z07:00", timestampStr.String)
	if parseErr == nil {
		return timestamp, nil
	}

	return time.Time{}, fmt.Errorf("failed to parse timestamp '%s': %v", timestampStr.String, parseErr)
}

// GetRiverData retrieves all river data from the database after a specific cutoff time
func (r *SQLiteRiverRepository) GetRiverData(cutoff time.Time) ([]entities.RiverData, error) {
	query := `
		SELECT id, river, station, water_level, water_temp, timestamp
		FROM river_data
		WHERE timestamp >= ?
		ORDER BY river, station, timestamp DESC`

	rows, err := r.db.Query(query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to query river data: %v", err)
	}
	defer rows.Close()

	var result []entities.RiverData
	for rows.Next() {
		var rd entities.RiverData
		if err := rows.Scan(
			&rd.ID,
			&rd.River,
			&rd.Station,
			&rd.WaterLevel,
			&rd.WaterTemp,
			&rd.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}
		result = append(result, rd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %v", err)
	}

	return result, nil
}
