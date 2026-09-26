// Command server runs the Kill Team tracker backend: it loads the rules
// census, builds the engine for the active team, and serves the JSON API.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"killteam/internal/api"
	"killteam/internal/engine"
	"killteam/internal/rules"
	"killteam/internal/store"
)

func main() {
	addr := flag.String("addr", envOr("KT_ADDR", ":8080"), "listen address")
	dataDir := flag.String("data", envOr("KT_DATA_DIR", defaultDataDir()), "directory of <teamId>.json rules files")
	gamesDir := flag.String("games", envOr("KT_GAMES_DIR", "games"), "directory for persisted game logs (empty to disable)")
	team := flag.String("team", envOr("KT_TEAM", ""), "active team id (default: the only one loaded)")
	static := flag.String("static", envOr("KT_STATIC_DIR", ""), "optional directory of built frontend assets to serve")
	flag.Parse()

	teams, err := rules.LoadAll(*dataDir)
	if err != nil {
		log.Fatalf("loading rules: %v", err)
	}

	// One engine per loaded team; games are routed to their team's engine.
	engines := make(map[string]*engine.Engine, len(teams))
	ids := make([]string, 0, len(teams))
	for id, d := range teams {
		engines[id] = engine.New(d)
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// The active team is the default for new games. It's selectable, but never
	// fatal: when several are loaded and none is chosen, pick deterministically.
	active := *team
	if active == "" {
		active = ids[0]
		if len(teams) > 1 {
			log.Printf("no -team given; defaulting active team to %q (loaded: %v)", active, ids)
		}
	}
	if _, ok := teams[active]; !ok {
		log.Fatalf("active team %q not found in %s (loaded: %v)", active, *dataDir, ids)
	}

	st, err := store.New(engines, *gamesDir)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	srv := api.New(st, teams, active)

	handler := srv.Handler()
	if *static != "" {
		handler = withStatic(handler, *static)
	}

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("kill-team tracker: team=%q teams=%d data=%s games=%s listening on %s",
		active, len(teams), *dataDir, dirLabel(*gamesDir), *addr)
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// withStatic serves built frontend files for non-API routes, falling back to
// index.html so the SPA can handle its own routing.
func withStatic(apiH http.Handler, dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	index := filepath.Join(dir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			apiH.ServeHTTP(w, r)
			return
		}
		if p := filepath.Join(dir, filepath.Clean(r.URL.Path)); fileExists(p) {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, index)
	})
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// defaultDataDir resolves the canonical rules directory whether the server is
// launched from the repo root or from backend/.
func defaultDataDir() string {
	for _, c := range []string{"data/teams", "../data/teams"} {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return "data/teams"
}

func dirLabel(d string) string {
	if d == "" {
		return "(in-memory)"
	}
	return d
}
