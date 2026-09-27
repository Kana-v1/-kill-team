// Package api exposes the engine and store over HTTP as a small JSON API.
//
// Routes:
//
//	GET  /api/teams                 list loaded teams
//	GET  /api/teams/{id}/rules      full rules census for one team
//	POST /api/games                 create a game -> {id, view}
//	GET  /api/games/{id}            current derived view
//	POST /api/games/{id}/events     append one event -> view
//	POST /api/games/{id}/undo       drop the last event -> view
//	POST /api/games/{id}/reset      clear the log -> view
//	GET  /api/games/{id}/log        raw event log (debug/export)
//	GET  /api/health                liveness
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"killteam/internal/engine"
	"killteam/internal/rules"
	"killteam/internal/store"
)

// Server wires the store, the active team's engine and the rules catalogue.
type Server struct {
	store    *store.Store
	teams    map[string]*rules.Data
	activeID string
}

// New builds a Server. teams is every loaded team keyed by id; store runs games
// for the active team.
func New(st *store.Store, teams map[string]*rules.Data, activeID string) *Server {
	return &Server{store: st, teams: teams, activeID: activeID}
}

// Handler returns the router with CORS applied.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/teams", s.listTeams)
	mux.HandleFunc("GET /api/teams/{id}/rules", s.teamRules)
	mux.HandleFunc("POST /api/games", s.createGame)
	mux.HandleFunc("GET /api/games/{id}", s.getGame)
	mux.HandleFunc("POST /api/games/{id}/events", s.postEvent)
	mux.HandleFunc("POST /api/games/{id}/undo", s.undo)
	mux.HandleFunc("POST /api/games/{id}/reset", s.reset)
	mux.HandleFunc("GET /api/games/{id}/log", s.log)
	return cors(mux)
}

type teamSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Edition string `json:"edition"`
	Active  bool   `json:"active"`
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) listTeams(w http.ResponseWriter, _ *http.Request) {
	out := make([]teamSummary, 0, len(s.teams))
	for id, d := range s.teams {
		out = append(out, teamSummary{ID: id, Name: d.Meta.Team, Edition: d.Meta.Edition, Active: id == s.activeID})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) teamRules(w http.ResponseWriter, r *http.Request) {
	d, ok := s.teams[r.PathValue("id")]
	if !ok {
		writeErr(w, http.StatusNotFound, "unknown team")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

type createRequest struct {
	Team string `json:"team"`
}

type createResponse struct {
	ID   string       `json:"id"`
	View *engine.View `json:"view"`
}

func (s *Server) createGame(w http.ResponseWriter, r *http.Request) {
	team := s.activeID
	if r.Body != nil {
		var req createRequest
		// A missing/empty body is fine — default to the active team.
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req); err == nil && req.Team != "" {
			team = req.Team
		}
	}
	if _, ok := s.teams[team]; !ok {
		writeErr(w, http.StatusBadRequest, "unknown team")
		return
	}
	id, err := s.store.Create(team)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not create game")
		return
	}
	view, err := s.store.State(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not derive new game")
		return
	}
	writeJSON(w, http.StatusCreated, createResponse{ID: id, View: view})
}

func (s *Server) getGame(w http.ResponseWriter, r *http.Request) {
	view, err := s.store.State(r.PathValue("id"))
	if s.handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) postEvent(w http.ResponseWriter, r *http.Request) {
	var ev engine.Event
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&ev); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid event body")
		return
	}
	if ev.T == "" {
		writeErr(w, http.StatusBadRequest, "event type required")
		return
	}
	view, err := s.store.Append(r.PathValue("id"), ev)
	if s.handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) undo(w http.ResponseWriter, r *http.Request) {
	view, err := s.store.Undo(r.PathValue("id"))
	if s.handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) reset(w http.ResponseWriter, r *http.Request) {
	view, err := s.store.Reset(r.PathValue("id"))
	if s.handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) log(w http.ResponseWriter, r *http.Request) {
	events, err := s.store.Log(r.PathValue("id"))
	if s.handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// handleStoreErr writes an error response for a store error and reports whether
// it did so.
func (s *Server) handleStoreErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "game not found")
		return true
	}
	writeErr(w, http.StatusInternalServerError, "internal error")
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// cors allows the Vite dev server (and any browser client) to call the API.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
