package db

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	conn *sqlx.DB
}

func OpenDatabase() (*DB, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("could not find config dir: %w", err)
	}

	appPath := filepath.Join(baseDir, "cvdl")
	dbPath := filepath.Join(appPath, "cvdl.db")

	err = os.MkdirAll(appPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("could not create directory: %w", err)
	}

	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	err = initSchema(db)
	if err != nil {
		return nil, err
	}
	return &DB{conn: db}, nil
}

func initSchema(db *sqlx.DB) error {
	const query = `
	CREATE TABLE IF NOT EXISTS downloads (
		video_id TEXT PRIMARY KEY,
		title TEXT
	);
	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT
	);
	`

	_, err := db.Exec(query)
	return err
}

func (db *DB) SetValue(key string, value string) error {
	query := `INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?);`
	_, err := db.conn.Exec(query, key, value)
	return err
}

func (db *DB) GetValue(key string) (string, error) {
	var value string
	query := `SELECT value FROM metadata WHERE key = ?;`
	err := db.conn.QueryRow(query, key).Scan(&value)
	return value, err
}

func (db *DB) RecordDownload(videoId int, title string) error {
	query := `INSERT OR REPLACE INTO downloads (video_id, title) VALUES (?, ?);`
	_, err := db.conn.Exec(query, videoId, title)
	return err
}

// returns true if `videoId` is downloaded
func (db *DB) CheckDownload(videoId int) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM downloads WHERE video_id = ?;`
	err := db.conn.QueryRow(query, videoId).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *DB) RemoveDownload(videoId string) error {
	query := `DELETE FROM downloads WHERE video_id = ?;`
	_, err := db.conn.Exec(query, videoId)
	return err
}

type Downloads struct {
	VideoId string `db:"video_id"`
	Title string
}

func (db *DB) ListDownload() []Downloads {
	downloads := []Downloads{}
	_ = db.conn.Select(&downloads, "SELECT * FROM downloads")
	return downloads
}
