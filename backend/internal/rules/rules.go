// Package rules holds the static rules data for a kill team and loads it from
// the canonical JSON census (data/teams/<id>.json).
//
// This is reference data only — effects, operatives, weapons, chapter tactics
// and the closed vocabularies. It says nothing about the current game; that is
// the engine's job. The structs mirror the JSON fields documented in CLAUDE.md.
package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Data is the whole census for one team.
type Data struct {
	Meta               Meta            `json:"meta"`
	Vocab              Vocab           `json:"vocab"`
	Effects            []Effect        `json:"effects"`
	Operatives         []Operative     `json:"operatives"`
	UniversalEquipment []Weapon        `json:"universalEquipment"`
	ChapterTactics     []ChapterTactic `json:"chapterTactics"`
}

// Meta carries provenance and the unresolved-questions list surfaced in Setup.
type Meta struct {
	Team       string   `json:"team"`
	TeamID     string   `json:"teamId"`
	Edition    string   `json:"edition"`
	Schema     int      `json:"schema"`
	Compiled   string   `json:"compiled"`
	Sources    []string `json:"sources"`
	Note       string   `json:"note"`
	Unresolved []string `json:"unresolved"`
	// SourcePdf and RulesVersion record which official document (and errata month) the census was checked against.
	SourcePdf    string `json:"sourcePdf,omitempty"`
	RulesVersion string `json:"rulesVersion,omitempty"`
	// DefaultRoster is the six operative instances a fresh game starts with.
	// Optional: the engine derives one (leader + specialists) when it is absent.
	DefaultRoster []string `json:"defaultRoster,omitempty"`
}

// Vocab is the closed set of legal windows, durations, triggers and the
// display contexts the UI offers as chips.
type Vocab struct {
	Windows   []string  `json:"windows"`
	Durations []string  `json:"durations"`
	Triggers  []string  `json:"triggers"`
	Contexts  []Context `json:"contexts"`
}

// Context is one selectable moment ("Shooting", "Counteracting", …).
type Context struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Cost is what an effect costs to activate.
type Cost struct {
	CP int `json:"cp"`
}

// Effect is a ploy, ability, equipment or faction rule — anything the tracker
// can surface. See CLAUDE.md "Data model" for field meanings.
type Effect struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Kind              string            `json:"kind"`
	Universal         bool              `json:"universal,omitempty"`
	Cost              Cost              `json:"cost"`
	Window            string            `json:"window,omitempty"`
	Windows           []string          `json:"windows,omitempty"`
	Duration          string            `json:"duration"`
	OncePer           string            `json:"once_per,omitempty"`
	RemindAt          []string          `json:"remind_at,omitempty"`
	Prompts           map[string]string `json:"prompts,omitempty"`
	Text              string            `json:"text"`
	Options           []Option          `json:"options,omitempty"`
	RequiresOperative string            `json:"requiresOperative,omitempty"`
	AlwaysOn          bool              `json:"alwaysOn,omitempty"`
	CostOverride      *Override         `json:"costOverride,omitempty"`
	CostOverrides     []Override        `json:"costOverrides,omitempty"`
	ChangesOptionOf   string            `json:"changesOptionOf,omitempty"`
	Requires          *Requires         `json:"requires,omitempty"`
	Source            string            `json:"source,omitempty"`
	Verify            []string          `json:"verify,omitempty"`
	Disputed          bool              `json:"disputed,omitempty"`
}

// Overrides returns the effect's cost overrides as a single slice, folding the
// legacy singular costOverride field into the list.
func (e Effect) Overrides() []Override {
	if len(e.CostOverrides) > 0 {
		return e.CostOverrides
	}
	if e.CostOverride != nil {
		return []Override{*e.CostOverride}
	}
	return nil
}

// Wins returns the windows an effect is legal in, folding the singular window
// field into the list form.
func (e Effect) Wins() []string {
	if len(e.Windows) > 0 {
		return e.Windows
	}
	if e.Window != "" {
		return []string{e.Window}
	}
	return nil
}

// Option is a sub-selection of an effect (Combat Doctrine's three doctrines).
type Option struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Condition string   `json:"condition,omitempty"`
	RemindAt  []string `json:"remind_at,omitempty"`
	Prompt    string   `json:"prompt,omitempty"`
}

// Override is a discount one effect grants another. See CLAUDE.md "Cost overrides".
type Override struct {
	Effect     string   `json:"effect,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	Options    []string `json:"options,omitempty"`
	Excludes   []string `json:"excludes,omitempty"`
	CP         int      `json:"cp"`
	OncePer    string   `json:"once_per,omitempty"`
	Group      string   `json:"group,omitempty"`
	SelectedIs string   `json:"selectedIs,omitempty"`
	Condition  string   `json:"condition,omitempty"`
}

// Requires is a precondition on an effect: a human-readable note and/or a
// dependency on another effect currently being active.
type Requires struct {
	Note         string `json:"note,omitempty"`
	ActiveEffect string `json:"activeEffect,omitempty"`
}

// Operative is one model on the roster.
type Operative struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Stats          Stats    `json:"stats"`
	Abilities      []string `json:"abilities"`
	Weapons        []Weapon `json:"weapons"`
	Icon           string   `json:"icon"`
	Accent         string   `json:"accent"`
	Photo          *string  `json:"photo"`
	Leader         bool     `json:"leader"`
	Multiple       bool     `json:"multiple"`
	Role           string   `json:"role"`
	ChapterVeteran bool     `json:"chapterVeteran"`
}

// Stats are the datacard header numbers.
type Stats struct {
	APL    int    `json:"apl"`
	Move   string `json:"move"`
	Save   string `json:"save"`
	Wounds int    `json:"wounds"`
}

// Weapon is one weapon line on a datacard (or a universal grenade).
type Weapon struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Atk   int    `json:"atk"`
	Hit   string `json:"hit"`
	Dmg   string `json:"dmg"`
	Rules string `json:"rules"`
}

// ChapterTactic is one selectable chapter tactic.
type ChapterTactic struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Text     string   `json:"text"`
	RemindAt []string `json:"remind_at"`
	Prompt   string   `json:"prompt"`
}

// Load reads and validates one team's census from dir/<id>.json.
func Load(dir, id string) (*Data, error) {
	path := filepath.Join(dir, id+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules %s: %w", path, err)
	}
	var d Data
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("parse rules %s: %w", path, err)
	}
	if len(d.Effects) == 0 || len(d.Operatives) == 0 {
		return nil, fmt.Errorf("rules %s look empty (%d effects, %d operatives)", path, len(d.Effects), len(d.Operatives))
	}
	return &d, nil
}

// LoadAll loads every team whose <id>.json sits in dir, keyed by team id.
func LoadAll(dir string) (map[string]*Data, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read teams dir %s: %w", dir, err)
	}
	out := map[string]*Data{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".json" {
			continue
		}
		id := name[:len(name)-len(".json")]
		d, err := Load(dir, id)
		if err != nil {
			return nil, err
		}
		out[id] = d
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no team json files in %s", dir)
	}
	return out, nil
}
