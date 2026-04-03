package server

import (
	"encoding/json"
	"log"
	"net/http"
	"github.com/stockyard-dev/stockyard-books/internal/store"
)

type Server struct { db *store.DB; mux *http.ServeMux }
func New(db *store.DB, limits Limits) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits}
	s.mux.HandleFunc("GET /api/accounts", s.listAccounts)
	s.mux.HandleFunc("POST /api/accounts", s.createAccount)
	s.mux.HandleFunc("GET /api/accounts/{id}", s.getAccount)
	s.mux.HandleFunc("PUT /api/accounts/{id}", s.updateAccount)
	s.mux.HandleFunc("DELETE /api/accounts/{id}", s.deleteAccount)
	s.mux.HandleFunc("GET /api/transactions", s.listTransactions)
	s.mux.HandleFunc("POST /api/transactions", s.createTransaction)
	s.mux.HandleFunc("DELETE /api/transactions/{id}", s.deleteTransaction)
	s.mux.HandleFunc("GET /api/pl", s.profitLoss)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
	return s
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func writeJSON(w http.ResponseWriter, code int, v any) { w.Header().Set("Content-Type","application/json"); w.WriteHeader(code); json.NewEncoder(w).Encode(v) }
func writeErr(w http.ResponseWriter, code int, msg string) { writeJSON(w, code, map[string]string{"error": msg}) }
func (s *Server) root(w http.ResponseWriter, r *http.Request) { if r.URL.Path != "/" { http.NotFound(w, r); return }; http.Redirect(w, r, "/ui", http.StatusFound) }
func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"accounts": orEmpty(s.db.ListAccounts())}) }
func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var a store.Account; json.NewDecoder(r.Body).Decode(&a)
	if a.Name == "" { writeErr(w, 400, "name required"); return }
	s.db.CreateAccount(&a); writeJSON(w, 201, s.db.GetAccount(a.ID))
}
func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	a := s.db.GetAccount(r.PathValue("id")); if a == nil { writeErr(w, 404, "not found"); return }; writeJSON(w, 200, a)
}
func (s *Server) updateAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id"); ex := s.db.GetAccount(id); if ex == nil { writeErr(w, 404, "not found"); return }
	var a store.Account; json.NewDecoder(r.Body).Decode(&a)
	if a.Name == "" { a.Name = ex.Name }; if a.Type == "" { a.Type = ex.Type }; if a.Currency == "" { a.Currency = ex.Currency }
	s.db.UpdateAccount(id, &a); writeJSON(w, 200, s.db.GetAccount(id))
}
func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) { s.db.DeleteAccount(r.PathValue("id")); writeJSON(w, 200, map[string]string{"deleted":"ok"}) }
func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]any{"transactions": orEmpty(s.db.ListTransactions(100))}) }
func (s *Server) createTransaction(w http.ResponseWriter, r *http.Request) {
	var t store.Transaction; json.NewDecoder(r.Body).Decode(&t)
	if t.Amount <= 0 { writeErr(w, 400, "amount required"); return }
	if t.DebitAcct == "" || t.CreditAcct == "" { writeErr(w, 400, "debit and credit accounts required"); return }
	s.db.CreateTransaction(&t); writeJSON(w, 201, t)
}
func (s *Server) deleteTransaction(w http.ResponseWriter, r *http.Request) { s.db.DeleteTransaction(r.PathValue("id")); writeJSON(w, 200, map[string]string{"deleted":"ok"}) }
func (s *Server) profitLoss(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.db.ProfitLoss()) }
func (s *Server) stats(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.db.Stats()) }
func (s *Server) health(w http.ResponseWriter, r *http.Request) { st := s.db.Stats(); writeJSON(w, 200, map[string]any{"status":"ok","service":"books","accounts":st.Accounts,"transactions":st.Transactions}) }
func orEmpty[T any](s []T) []T { if s == nil { return []T{} }; return s }
func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }
