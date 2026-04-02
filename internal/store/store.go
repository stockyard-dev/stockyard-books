package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct { db *sql.DB }

type Transaction struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Amount       string   `json:"amount"`
	Category     string   `json:"category"`
	Type         string   `json:"type"`
	Date         string   `json:"date"`
	CreatedAt    string   `json:"created_at"`
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	dsn := filepath.Join(dataDir, "books.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,\n\t\t\tdescription TEXT DEFAULT '',\n\t\t\tamount TEXT DEFAULT '',\n\t\t\tcategory TEXT DEFAULT '',\n\t\t\ttype TEXT DEFAULT 'expense',\n\t\t\tdate TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now'))
		)`)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

func (d *DB) Create(e *Transaction) error {
	e.ID = genID()
	e.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err := d.db.Exec(`INSERT INTO transactions (id, description, amount, category, type, date, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Description, e.Amount, e.Category, e.Type, e.Date, e.CreatedAt)
	return err
}

func (d *DB) Get(id string) *Transaction {
	row := d.db.QueryRow(`SELECT id, description, amount, category, type, date, created_at FROM transactions WHERE id=?`, id)
	var e Transaction
	if err := row.Scan(&e.ID, &e.Description, &e.Amount, &e.Category, &e.Type, &e.Date, &e.CreatedAt); err != nil {
		return nil
	}
	return &e
}

func (d *DB) List() []Transaction {
	rows, err := d.db.Query(`SELECT id, description, amount, category, type, date, created_at FROM transactions ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []Transaction
	for rows.Next() {
		var e Transaction
		if err := rows.Scan(&e.ID, &e.Description, &e.Amount, &e.Category, &e.Type, &e.Date, &e.CreatedAt); err != nil {
			continue
		}
		result = append(result, e)
	}
	return result
}

func (d *DB) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM transactions WHERE id=?`, id)
	return err
}

func (d *DB) Count() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&n)
	return n
}
