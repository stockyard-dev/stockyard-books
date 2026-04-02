package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

type Account struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"` // asset, liability, equity, revenue, expense
	Currency  string  `json:"currency"`
	Balance   float64 `json:"balance"`
	CreatedAt string  `json:"created_at"`
}

type Transaction struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	DebitAcct   string  `json:"debit_account_id"`
	CreditAcct  string  `json:"credit_account_id"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category,omitempty"`
	Reference   string  `json:"reference,omitempty"`
	CreatedAt   string  `json:"created_at"`
	DebitName   string  `json:"debit_name,omitempty"`
	CreditName  string  `json:"credit_name,omitempty"`
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil { return nil, err }
	dsn := filepath.Join(dataDir, "books.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil { return nil, err }
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS accounts (id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT DEFAULT 'asset', currency TEXT DEFAULT 'USD', created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS transactions (id TEXT PRIMARY KEY, date TEXT NOT NULL, description TEXT DEFAULT '', debit_account_id TEXT NOT NULL, credit_account_id TEXT NOT NULL, amount REAL NOT NULL, category TEXT DEFAULT '', reference TEXT DEFAULT '', created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE INDEX IF NOT EXISTS idx_txn_date ON transactions(date)`,
		`CREATE INDEX IF NOT EXISTS idx_txn_debit ON transactions(debit_account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_txn_credit ON transactions(credit_account_id)`,
	} {
		if _, err := db.Exec(q); err != nil { return nil, fmt.Errorf("migrate: %w", err) }
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string { return time.Now().UTC().Format(time.RFC3339) }
func today() string { return time.Now().Format("2006-01-02") }

func (d *DB) calcBalance(acctID string) float64 {
	var debits, credits float64
	d.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM transactions WHERE debit_account_id=?`, acctID).Scan(&debits)
	d.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM transactions WHERE credit_account_id=?`, acctID).Scan(&credits)
	return debits - credits
}

func (d *DB) CreateAccount(a *Account) error {
	a.ID = genID(); a.CreatedAt = now()
	if a.Type == "" { a.Type = "asset" }; if a.Currency == "" { a.Currency = "USD" }
	_, err := d.db.Exec(`INSERT INTO accounts (id,name,type,currency,created_at) VALUES (?,?,?,?,?)`, a.ID, a.Name, a.Type, a.Currency, a.CreatedAt)
	return err
}

func (d *DB) GetAccount(id string) *Account {
	var a Account
	if err := d.db.QueryRow(`SELECT id,name,type,currency,created_at FROM accounts WHERE id=?`, id).Scan(&a.ID, &a.Name, &a.Type, &a.Currency, &a.CreatedAt); err != nil { return nil }
	a.Balance = d.calcBalance(a.ID); return &a
}

func (d *DB) ListAccounts() []Account {
	rows, _ := d.db.Query(`SELECT id,name,type,currency,created_at FROM accounts ORDER BY type, name`)
	if rows == nil { return nil }; defer rows.Close()
	var out []Account
	for rows.Next() {
		var a Account; rows.Scan(&a.ID, &a.Name, &a.Type, &a.Currency, &a.CreatedAt)
		a.Balance = d.calcBalance(a.ID); out = append(out, a)
	}
	return out
}

func (d *DB) UpdateAccount(id string, a *Account) error {
	_, err := d.db.Exec(`UPDATE accounts SET name=?,type=?,currency=? WHERE id=?`, a.Name, a.Type, a.Currency, id); return err
}

func (d *DB) DeleteAccount(id string) error {
	_, err := d.db.Exec(`DELETE FROM accounts WHERE id=?`, id); return err
}

func (d *DB) CreateTransaction(t *Transaction) error {
	t.ID = genID(); t.CreatedAt = now()
	if t.Date == "" { t.Date = today() }
	_, err := d.db.Exec(`INSERT INTO transactions (id,date,description,debit_account_id,credit_account_id,amount,category,reference,created_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Date, t.Description, t.DebitAcct, t.CreditAcct, t.Amount, t.Category, t.Reference, t.CreatedAt)
	return err
}

func (d *DB) ListTransactions(limit int) []Transaction {
	if limit <= 0 { limit = 50 }
	rows, _ := d.db.Query(`SELECT t.id,t.date,t.description,t.debit_account_id,t.credit_account_id,t.amount,t.category,t.reference,t.created_at,COALESCE(d.name,''),COALESCE(c.name,'') FROM transactions t LEFT JOIN accounts d ON t.debit_account_id=d.id LEFT JOIN accounts c ON t.credit_account_id=c.id ORDER BY t.date DESC, t.created_at DESC LIMIT ?`, limit)
	if rows == nil { return nil }; defer rows.Close()
	var out []Transaction
	for rows.Next() {
		var t Transaction
		rows.Scan(&t.ID, &t.Date, &t.Description, &t.DebitAcct, &t.CreditAcct, &t.Amount, &t.Category, &t.Reference, &t.CreatedAt, &t.DebitName, &t.CreditName)
		out = append(out, t)
	}
	return out
}

func (d *DB) DeleteTransaction(id string) error { _, err := d.db.Exec(`DELETE FROM transactions WHERE id=?`, id); return err }

type PL struct {
	Revenue  float64 `json:"revenue"`
	Expenses float64 `json:"expenses"`
	Net      float64 `json:"net"`
}

func (d *DB) ProfitLoss() PL {
	var pl PL
	d.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM transactions WHERE debit_account_id IN (SELECT id FROM accounts WHERE type='revenue')`).Scan(&pl.Revenue)
	d.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM transactions WHERE debit_account_id IN (SELECT id FROM accounts WHERE type='expense')`).Scan(&pl.Expenses)
	pl.Net = pl.Revenue - pl.Expenses
	return pl
}

type Stats struct { Accounts int `json:"accounts"`; Transactions int `json:"transactions"` }
func (d *DB) Stats() Stats {
	var s Stats
	d.db.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&s.Accounts)
	d.db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&s.Transactions)
	return s
}
