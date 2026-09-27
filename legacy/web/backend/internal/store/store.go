// Package store keeps the append-only event log for each game and mediates
// every change through that game's team engine, so callers only ever see
// derived views.
//
// A game is bound to one team for its whole life; the store holds an engine per
// team and routes each game to its own.
//
// Persistence is best-effort, matching the original app's localStorage stance:
// logs are mirrored to <dir>/<id>.json, but a failed read or write must never
// take the server down.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"killteam/internal/engine"
)

// ErrNotFound is returned when a game id is unknown.
var ErrNotFound = errors.New("game not found")

// ErrUnknownTeam is returned when a game is requested for a team with no engine.
var ErrUnknownTeam = errors.New("unknown team")

// game is one game's team binding and event log.
type game struct {
	Team   string         `json:"team"`
	Events []engine.Event `json:"events"`
}

// Store holds every game in memory, guarded by a mutex, with an optional on-disk
// mirror. It is safe for concurrent use.
type Store struct {
	mu      sync.Mutex
	engines map[string]*engine.Engine
	games   map[string]*game
	dir     string // "" disables persistence
}

// New builds a Store over the given per-team engines. If dir is non-empty, logs
// already present there are loaded and future changes are mirrored back.
func New(engines map[string]*engine.Engine, dir string) (*Store, error) {
	s := &Store{engines: engines, games: map[string]*game{}, dir: dir}
	if dir == "" {
		return s, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		// Persistence is best-effort: run in memory rather than fail.
		s.dir = ""
		return s, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return s, nil
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		id := e.Name()[:len(e.Name())-len(".json")]
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var g game
		if json.Unmarshal(raw, &g) != nil || g.Team == "" {
			continue
		}
		if _, ok := s.engines[g.Team]; !ok {
			continue // team no longer loaded; skip rather than crash
		}
		if g.Events == nil {
			g.Events = []engine.Event{}
		}
		s.games[id] = &g
	}
	return s, nil
}

// Create makes a new empty game for the given team and returns its id.
func (s *Store) Create(team string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.engines[team]; !ok {
		return "", ErrUnknownTeam
	}
	id, err := newID()
	if err != nil {
		return "", err
	}
	s.games[id] = &game{Team: team, Events: []engine.Event{}}
	s.persist(id)
	return id, nil
}

// State derives the current view for a game. Surfacing (marking that a paid
// effect was shown at a matching moment) is the one render-time side effect the
// original app performs; it is preserved here so the end-of-turning-point
// summary can find effects that were never used.
func (s *Store) State(id string) (*engine.View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deriveLocked(id)
}

// Append adds one event, then derives. Returns the resulting view.
func (s *Store) Append(id string, ev engine.Event) (*engine.View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[id]
	if !ok {
		return nil, ErrNotFound
	}
	g.Events = append(g.Events, ev)
	return s.deriveLocked(id)
}

// Undo drops the last event, skipping trailing SURFACE markers first (they are
// bookkeeping, not user actions), then re-derives.
func (s *Store) Undo(id string) (*engine.View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[id]
	if !ok {
		return nil, ErrNotFound
	}
	log := g.Events
	for len(log) > 0 && log[len(log)-1].T == engine.EvSurface {
		log = log[:len(log)-1]
	}
	if len(log) > 0 {
		log = log[:len(log)-1]
	}
	g.Events = log
	return s.deriveLocked(id)
}

// Reset clears the whole log (a fresh game under the same id and team).
func (s *Store) Reset(id string) (*engine.View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[id]
	if !ok {
		return nil, ErrNotFound
	}
	g.Events = []engine.Event{}
	return s.deriveLocked(id)
}

// Log returns a copy of the raw event log (for debugging / export).
func (s *Store) Log(id string) ([]engine.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[id]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]engine.Event(nil), g.Events...), nil
}

// deriveLocked folds, derives, appends any fresh SURFACE markers, persists, and
// returns the view. Caller must hold s.mu.
func (s *Store) deriveLocked(id string) (*engine.View, error) {
	g, ok := s.games[id]
	if !ok {
		return nil, ErrNotFound
	}
	eng, ok := s.engines[g.Team]
	if !ok {
		return nil, ErrUnknownTeam
	}
	state := eng.Fold(g.Events)
	view := eng.Derive(state, len(g.Events))
	view.Team = g.Team // the route key the client uses to fetch this team's rules
	if fresh := view.FreshSurfaces(); len(fresh) > 0 {
		for _, eid := range fresh {
			g.Events = append(g.Events, engine.Event{T: engine.EvSurface, P: engine.Params{ID: eid}})
		}
	}
	s.persist(id)
	return view, nil
}

// persist mirrors a game's log to disk. Best-effort: errors are swallowed.
func (s *Store) persist(id string) {
	if s.dir == "" {
		return
	}
	raw, err := json.Marshal(s.games[id])
	if err != nil {
		return
	}
	tmp := filepath.Join(s.dir, id+".json.tmp")
	final := filepath.Join(s.dir, id+".json")
	if os.WriteFile(tmp, raw, 0o644) == nil {
		_ = os.Rename(tmp, final)
	}
}

func newID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
