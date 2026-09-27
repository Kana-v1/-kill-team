// Package engine is the canonical rules engine: the append-only event log is
// folded into a State, and State + rules are derived into the view the client
// renders. It is a direct, tested port of the original client-side reducer.
//
// Two invariants carry over from CLAUDE.md and must not be broken:
//
//  1. Derive, never mutate. Expiry is not an event — an effect simply stops
//     matching once the turning-point counter moves. There is no
//     subtract-on-expiry anywhere.
//  2. The log is the state. Every change is an event; current state is a pure
//     function of the log. Undo is "drop the last event and re-fold".
package engine

import (
	"strconv"

	"killteam/internal/rules"
)

// Event is one entry in the append-only log.
type Event struct {
	T string `json:"t"`
	P Params `json:"p"`
}

// Params carries the (sparse) payload of an event. Only the fields relevant to
// a given event type are set.
type Params struct {
	Ctx  string `json:"ctx,omitempty"`
	D    int    `json:"d,omitempty"`
	ID   string `json:"id,omitempty"`
	Opt  string `json:"opt,omitempty"`
	Slot string `json:"slot,omitempty"`
	K    string `json:"k,omitempty"`
	// Ini is who has initiative for the new turning point (TP_NEXT): "us" or "them".
	Ini string `json:"ini,omitempty"`
}

// Event type constants — the vocabulary of the log.
const (
	EvCtx          = "CTX"
	EvCP           = "CP"
	EvTPNext       = "TP_NEXT"
	EvTPPrev       = "TP_PREV"
	EvTPReset      = "TP_RESET"
	EvActivate     = "ACTIVATE"
	EvEnd          = "END"
	EvUseBattle    = "USE_BATTLE"
	EvDismiss      = "DISMISS"
	EvSurface      = "SURFACE"
	EvOp           = "OP"
	EvDown         = "DOWN"
	EvLeader       = "LEADER"
	EvRoster       = "ROSTER"
	EvCount        = "COUNT"
	EvEquip        = "EQUIP"
	EvTactic       = "TACTIC"
	EvClearSummary = "CLEAR_SUMMARY"
)

// ActiveEntry is an effect currently running, recorded when it was activated.
type ActiveEntry struct {
	ID  string `json:"id"`
	Opt string `json:"opt"`
	TP  int    `json:"tp"`
	Seq int    `json:"seq"`
}

// Summary is the end-of-turning-point "never used" reminder.
type Summary struct {
	TP    int      `json:"tp"`
	Names []string `json:"names"`
}

// State is the folded result of the event log. It is game bookkeeping only:
// no board, no dice, no rules resolution.
type State struct {
	TP          int               `json:"tp"`
	CP          int               `json:"cp"`
	Ctx         string            `json:"ctx"`
	Roster      []string          `json:"roster"`
	Op          string            `json:"op"`
	Dead        map[string]bool   `json:"dead"`
	Equip       map[string]bool   `json:"equip"`
	Tactics     map[string]string `json:"tactics"`
	Active      []ActiveEntry     `json:"active"`
	Used        map[string]int    `json:"used"`
	UsedBattle  map[string]bool   `json:"usedBattle"`
	UsedTp      map[string]int    `json:"usedTp"`
	Dismissed   map[string]bool   `json:"dismissed"`
	Surfaced    map[string]bool   `json:"surfaced"`
	LastSummary *Summary          `json:"lastSummary"`
	Seq         int               `json:"seq"`
}

// shortDurations expire the moment the context changes.
var shortDurations = map[string]bool{
	"instant":            true,
	"this_sequence":      true,
	"this_activation":    true,
	"this_counteraction": true,
}

// Engine binds the rules data so the pure fold/derive functions can consult it.
type Engine struct {
	Rules       *rules.Data
	byID        map[string]*rules.Effect
	opByID      map[string]*rules.Operative
	leaders     map[string]bool
	defaultRost []string
}

// New builds an Engine with lookup tables over the rules data.
func New(r *rules.Data) *Engine {
	e := &Engine{
		Rules:   r,
		byID:    make(map[string]*rules.Effect, len(r.Effects)),
		opByID:  make(map[string]*rules.Operative, len(r.Operatives)),
		leaders: map[string]bool{},
	}
	for i := range r.Effects {
		e.byID[r.Effects[i].ID] = &r.Effects[i]
	}
	for i := range r.Operatives {
		e.opByID[r.Operatives[i].ID] = &r.Operatives[i]
		if r.Operatives[i].Leader {
			e.leaders[r.Operatives[i].ID] = true
		}
	}
	e.defaultRost = e.resolveDefaultRoster()
	return e
}

// resolveDefaultRoster is the team's declared starting six, or a derived one
// (its leader followed by non-leader operatives) when none is declared. This is
// what keeps the engine team-agnostic — nothing here is hardcoded to a team.
func (e *Engine) resolveDefaultRoster() []string {
	if len(e.Rules.Meta.DefaultRoster) > 0 {
		return append([]string(nil), e.Rules.Meta.DefaultRoster...)
	}
	var leader string
	var others []string
	for i := range e.Rules.Operatives {
		o := &e.Rules.Operatives[i]
		if o.Leader && leader == "" {
			leader = o.ID
		} else if !o.Leader {
			others = append(others, o.ID)
		}
	}
	out := []string{}
	if leader != "" {
		out = append(out, leader)
	}
	for _, id := range others {
		if len(out) >= 6 {
			break
		}
		out = append(out, id)
	}
	return out
}

// typeOf strips the "#n" instance suffix: "intercessor_warrior#2" -> base.
func typeOf(inst string) string {
	for i := 0; i < len(inst); i++ {
		if inst[i] == '#' {
			return inst[:i]
		}
	}
	return inst
}

func (e *Engine) effect(id string) *rules.Effect { return e.byID[id] }

func (e *Engine) opFor(inst string) *rules.Operative { return e.opByID[typeOf(inst)] }

func (e *Engine) isLeader(inst string) bool { return e.leaders[typeOf(inst)] }

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// blank is the starting state before any events are applied.
func (e *Engine) blank() *State {
	roster := append([]string(nil), e.defaultRost...)
	op := ""
	if len(roster) > 0 {
		op = roster[0]
	}
	return &State{
		// Core rules: 2CP at the start, +1 in the first Strategy phase.
		TP: 1, CP: 3, Ctx: "strategy_phase",
		Roster:     roster,
		Op:         op,
		Dead:       map[string]bool{},
		Equip:      map[string]bool{},
		Tactics:    map[string]string{"primary": "", "secondary": "", "extra": ""},
		Active:     []ActiveEntry{},
		Used:       map[string]int{},
		UsedBattle: map[string]bool{},
		UsedTp:     map[string]int{},
		Dismissed:  map[string]bool{},
		Surfaced:   map[string]bool{},
	}
}

// clearTp resets the per-turning-point bookkeeping.
func clearTp(s *State) {
	s.Active = []ActiveEntry{}
	s.Used = map[string]int{}
	s.UsedTp = map[string]int{}
	s.Dismissed = map[string]bool{}
	s.Surfaced = map[string]bool{}
	s.Ctx = "strategy_phase"
}

// Fold reduces the whole log to a State.
func (e *Engine) Fold(events []Event) *State {
	s := e.blank()
	for i := range events {
		e.apply(s, events[i])
	}
	return s
}

// apply mutates s by one event. This is the only place state is mutated.
func (e *Engine) apply(s *State, ev Event) {
	p := ev.P
	switch ev.T {
	case EvCtx:
		s.Ctx = p.Ctx
		kept := s.Active[:0]
		for _, a := range s.Active {
			if ef := e.effect(a.ID); ef == nil || !shortDurations[ef.Duration] {
				kept = append(kept, a)
			}
		}
		s.Active = append([]ActiveEntry(nil), kept...)

	case EvCP:
		s.CP = maxInt(0, s.CP+p.D)

	case EvTPNext:
		var orphans []string
		for _, a := range s.Active {
			ef := e.effect(a.ID)
			if ef != nil && ef.Duration == "end_of_turning_point" && !s.Surfaced[a.ID] {
				orphans = append(orphans, ef.Name)
			}
		}
		if len(orphans) > 0 {
			s.LastSummary = &Summary{TP: s.TP, Names: orphans}
		} else {
			s.LastSummary = nil
		}
		s.TP++
		// Core rules: after the first Strategy phase, the player with initiative
		// gains 1CP and the player without gains 2CP.
		if p.Ini == "them" {
			s.CP += 2
		} else {
			s.CP++
		}
		clearTp(s)

	case EvTPPrev:
		if s.TP > 1 {
			s.TP--
			s.LastSummary = nil
		}

	case EvTPReset:
		clearTp(s)
		s.LastSummary = nil

	case EvActivate:
		ef := e.effect(p.ID)
		if ef == nil {
			return
		}
		q := e.quote(s, ef, p.Opt)
		s.CP = maxInt(0, s.CP-q.CP)
		s.Used[ef.ID] = s.TP
		if q.Key != "" && q.Once == "battle" {
			s.UsedBattle[q.Key] = true
		}
		if q.Key != "" && q.Once == "turning_point" {
			s.UsedTp[q.Key] = s.TP
		}
		if ef.ChangesOptionOf != "" {
			for i := range s.Active {
				if s.Active[i].ID == ef.ChangesOptionOf {
					s.Active[i].Opt = p.Opt
				}
			}
		} else if ef.Duration != "instant" {
			s.Seq++
			s.Active = append(s.Active, ActiveEntry{ID: ef.ID, Opt: p.Opt, TP: s.TP, Seq: s.Seq})
		}

	case EvEnd:
		kept := s.Active[:0]
		for _, a := range s.Active {
			if a.ID != p.ID {
				kept = append(kept, a)
			}
		}
		s.Active = append([]ActiveEntry(nil), kept...)

	case EvUseBattle:
		s.UsedBattle[p.ID] = true

	case EvDismiss:
		s.Dismissed[p.K] = true

	case EvSurface:
		s.Surfaced[p.ID] = true

	case EvOp:
		s.Op = p.ID

	case EvDown:
		s.Dead[p.ID] = !s.Dead[p.ID]

	case EvLeader:
		had := false
		for _, x := range s.Roster {
			if typeOf(x) == p.ID {
				had = true
			}
		}
		kept := s.Roster[:0]
		for _, x := range s.Roster {
			if e.isLeader(x) {
				delete(s.Dead, x)
			} else {
				kept = append(kept, x)
			}
		}
		s.Roster = append([]string(nil), kept...)
		if !had {
			s.Roster = append([]string{p.ID}, s.Roster...)
		}
		e.ensureOp(s)

	case EvRoster:
		if contains(s.Roster, p.ID) {
			s.Roster = removeString(s.Roster, p.ID)
		} else {
			s.Roster = append(s.Roster, p.ID)
		}
		e.ensureOp(s)

	case EvCount:
		var mine []string
		for _, x := range s.Roster {
			if typeOf(x) == p.ID {
				mine = append(mine, x)
			}
		}
		if p.D > 0 {
			n := 1
			for contains(s.Roster, p.ID+"#"+strconv.Itoa(n)) {
				n++
			}
			s.Roster = append(s.Roster, p.ID+"#"+strconv.Itoa(n))
		} else if len(mine) > 0 {
			drop := mine[len(mine)-1]
			s.Roster = removeString(s.Roster, drop)
			delete(s.Dead, drop)
		}
		e.ensureOp(s)

	case EvEquip:
		s.Equip[p.ID] = !s.Equip[p.ID]

	case EvTactic:
		on := s.Tactics[p.Slot] == p.ID
		if on {
			s.Tactics[p.Slot] = ""
		} else {
			s.Tactics[p.Slot] = p.ID
			for _, other := range []string{"primary", "secondary", "extra"} {
				if other != p.Slot && s.Tactics[other] == p.ID {
					s.Tactics[other] = ""
				}
			}
		}

	case EvClearSummary:
		s.LastSummary = nil
	}
}

// ensureOp keeps the selected operative valid after a roster change.
func (e *Engine) ensureOp(s *State) {
	if !contains(s.Roster, s.Op) {
		if len(s.Roster) > 0 {
			s.Op = s.Roster[0]
		} else {
			s.Op = ""
		}
	}
}

func removeString(s []string, v string) []string {
	out := make([]string, 0, len(s))
	for _, x := range s {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}
