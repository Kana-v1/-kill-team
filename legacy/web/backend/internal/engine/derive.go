package engine

import (
	"sort"
	"strconv"
	"strings"

	"killteam/internal/rules"
)

// ctxTriggers maps a display context to the trigger moments it fires.
var ctxTriggers = map[string][]string{
	"strategy_phase":   {"strategy_phase", "turning_point.start"},
	"activation.own":   {"activation.start"},
	"combat.shoot":     {"combat.shoot.before_roll", "combat.shoot.after_roll"},
	"combat.fight":     {"combat.fight.before_roll", "combat.fight.after_roll"},
	"combat.retaliate": {"combat.retaliate.before_roll", "combat.retaliate.after_roll"},
	"defence.shooting": {"defence.shooting.before_roll", "defence.shooting.after_roll"},
	"counteract":       {"counteract.available"},
}

var kindLabels = map[string]string{
	"strategy_ploy":     "Strategy ploy",
	"firefight_ploy":    "Firefight ploy",
	"equipment":         "Equipment",
	"faction_rule":      "Faction rule",
	"operative_ability": "Ability",
	"chapter_tactic":    "Chapter tactic",
}

var expiryLabels = map[string]string{
	"end_of_turning_point": "expires end of turning point",
	"this_sequence":        "expires end of this sequence",
	"this_activation":      "expires end of this activation",
	"this_counteraction":   "expires end of this counteraction",
	"battle":               "lasts all battle",
}

// route is a single cost override that could apply to an effect right now.
type route struct {
	CP        int
	Condition string
	From      string
	Options   []string
	Key       string
	Once      string
	Disputed  bool
	Inactive  bool
	Needs     string
}

// quoted is the price to show for an effect, and why.
type quoted struct {
	CP        int
	Condition string
	From      string
	Key       string
	Once      string
	Maybe     []route
}

// present reports whether an effect exists in the current game: its operative
// is on the table (and not incapacitated), or, for equipment, it was taken.
func (e *Engine) present(s *State, ef *rules.Effect) bool {
	if ef.Kind == "equipment" && !s.Equip[ef.ID] {
		return false
	}
	if ef.RequiresOperative == "" {
		return true
	}
	for _, inst := range s.Roster {
		if typeOf(inst) == ef.RequiresOperative && !s.Dead[inst] {
			return true
		}
	}
	return false
}

// routes collects every cost override that could apply to ef right now.
func (e *Engine) routes(s *State, ef *rules.Effect, optID string) []route {
	var out []route
	for i := range e.Rules.Effects {
		p := &e.Rules.Effects[i]
		if !e.present(s, p) {
			continue
		}
		if p.Kind == "equipment" && s.Used[p.ID] == s.TP {
			continue
		}
		for _, ov := range p.Overrides() {
			if ov.Effect != "" && ov.Effect != ef.ID {
				continue
			}
			if ov.Kind != "" && ov.Kind != ef.Kind {
				continue
			}
			if contains(ov.Excludes, ef.ID) {
				continue
			}
			if len(ov.Options) > 0 && optID != "" && !contains(ov.Options, optID) {
				continue
			}
			wrongOperative := ov.SelectedIs != "" && typeOf(s.Op) != ov.SelectedIs
			// a shared group means one use covers every option of that ability
			scope := ov.Group
			if scope == "" {
				if len(ov.Options) > 0 {
					o := optID
					if o == "" {
						o = "*"
					}
					scope = ef.ID + "|" + o
				} else {
					scope = ef.ID + "|*"
				}
			}
			key := ""
			if ov.OncePer != "" {
				key = p.ID + "|" + scope
			}
			if key != "" && ov.OncePer == "battle" && s.UsedBattle[key] {
				continue
			}
			if key != "" && ov.OncePer == "turning_point" && s.UsedTp[key] == s.TP {
				continue
			}
			out = append(out, route{
				CP: ov.CP, Condition: ov.Condition, From: p.Name, Options: ov.Options,
				Key: key, Once: ov.OncePer, Disputed: p.Disputed, Inactive: wrongOperative,
				Needs: ov.SelectedIs,
			})
		}
	}
	return out
}

// quote returns the price to show and the reason. Option-specific routes do not
// move the headline; they surface as "can be free" hints instead.
func (e *Engine) quote(s *State, ef *rules.Effect, optID string) quoted {
	all := e.routes(s, ef, optID)
	var maybe []route
	best := quoted{CP: ef.Cost.CP}
	for _, r := range all {
		firm := !r.Inactive && (len(r.Options) == 0 || (optID != "" && contains(r.Options, optID)))
		if !firm {
			maybe = append(maybe, r)
			continue
		}
		if r.CP < best.CP {
			best = quoted{CP: r.CP, Condition: r.Condition, From: r.From, Key: r.Key, Once: r.Once}
		}
	}
	best.Maybe = maybe
	return best
}

// ---- view models the thin client renders ----

// View is the full derived snapshot returned to the client.
type View struct {
	Team        string            `json:"team"`
	TP          int               `json:"tp"`
	CP          int               `json:"cp"`
	Ctx         string            `json:"ctx"`
	Roster      []string          `json:"roster"`
	Op          string            `json:"op"`
	Dead        map[string]bool   `json:"dead"`
	Equip       map[string]bool   `json:"equip"`
	Tactics     map[string]string `json:"tactics"`
	CanUndo     bool              `json:"canUndo"`
	CanPrevTP   bool              `json:"canPrevTP"`
	Roster6     RosterStatus      `json:"rosterStatus"`
	Veterans    []string          `json:"veterans"`
	Prompts     []Prompt          `json:"prompts"`
	Summary     *Summary          `json:"summary"`
	Usable      []UsableCard      `json:"usable"`
	Running     []RunningCard     `json:"running"`
	Spent       []SpentCard       `json:"spent"`
	VerifyNote  VerifyNote        `json:"verifyNote"`
	freshPrompt []string          // paid prompts not yet surfaced (server-side only)
}

// FreshSurfaces are paid prompts shown for the first time this window; the store
// appends a SURFACE event for each so the end-of-TP summary can find them.
func (v *View) FreshSurfaces() []string { return v.freshPrompt }

// RosterStatus summarises Setup validity.
type RosterStatus struct {
	Total   int  `json:"total"`
	Leaders int  `json:"leaders"`
	OK      bool `json:"ok"`
}

// VerifyNote drives the Setup panel's sourcing caveat.
type VerifyNote struct {
	Flagged    int      `json:"flagged"`
	Unresolved []string `json:"unresolved"`
}

// Prompt is one amber banner reminder.
type Prompt struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Text string `json:"text"`
	Key  string `json:"key"`
	Paid bool   `json:"paid"`
}

// MaybeRoute is a dim "can be free" hint under a usable card.
type MaybeRoute struct {
	From      string   `json:"from"`
	Options   []string `json:"options,omitempty"`
	Condition string   `json:"condition"`
	Needs     bool     `json:"needs"`
	Disputed  bool     `json:"disputed"`
}

// OptionQuote is a priced sub-selection shown when a card is expanded.
type OptionQuote struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Condition string `json:"condition"`
	CP        int    `json:"cp"`
	Free      bool   `json:"free"`
}

// UsableCard is an activatable effect in the current window.
type UsableCard struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Kind      string        `json:"kind"`
	KindLabel string        `json:"kindLabel"`
	Text      string        `json:"text"`
	CostBase  int           `json:"costBase"`
	CostShown string        `json:"costShown"`
	Reduced   bool          `json:"reduced"`
	Free      bool          `json:"free"`
	Afford    bool          `json:"afford"`
	Condition string        `json:"condition,omitempty"`
	From      string        `json:"from,omitempty"`
	Maybe     []MaybeRoute  `json:"maybe,omitempty"`
	Options   []OptionQuote `json:"options,omitempty"`
}

// RunningCard is an effect currently in play (or always-on).
type RunningCard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	KindLabel   string `json:"kindLabel"`
	Text        string `json:"text"`
	Meta        string `json:"meta"`
	Now         bool   `json:"now"`
	Always      bool   `json:"always"`
	CanEnd      bool   `json:"canEnd"`
	CanMarkUsed bool   `json:"canMarkUsed"`
	Disputed    bool   `json:"disputed"`
}

// SpentCard is an effect already used this turning point.
type SpentCard struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	KindLabel string `json:"kindLabel"`
}

func (e *Engine) kindLabel(ef *rules.Effect, slot string) string {
	k := kindLabels[ef.Kind]
	if k == "" {
		k = ef.Kind
	}
	if ef.Universal {
		k = "Universal " + strings.ToLower(k)
	}
	if slot != "" {
		k += " · " + slot
	}
	if ef.RequiresOperative != "" {
		if o := e.opByID[ef.RequiresOperative]; o != nil {
			k += " · " + o.Name
		}
	}
	return k
}

// runningItem couples a running effect with its display slot/opt.
type runningItem struct {
	ef     *rules.Effect
	opt    string
	slot   string
	always bool
	now    bool
}

// fires reports whether a running item should surface in the current window.
func fires(ri runningItem, triggers []string) bool {
	at := ri.ef.RemindAt
	if ri.opt != "" {
		for _, o := range ri.ef.Options {
			if o.ID == ri.opt && len(o.RemindAt) > 0 {
				at = o.RemindAt
				break
			}
		}
	}
	for _, t := range at {
		if contains(triggers, t) {
			return true
		}
	}
	return false
}

// Derive turns State + rules into the view the client renders. Pure: it never
// mutates s. The freshPrompt list is advisory output, not a state change.
func (e *Engine) Derive(s *State, logLen int) *View {
	triggers := ctxTriggers[s.Ctx]
	activeIDs := map[string]bool{}
	for _, a := range s.Active {
		activeIDs[a.ID] = true
	}

	// Initialise as empty (never nil) so the JSON always carries arrays the thin
	// client can map over directly.
	usable := []UsableCard{}
	spent := []SpentCard{}
	var running []runningItem

	for i := range e.Rules.Effects {
		ef := &e.Rules.Effects[i]
		if !e.present(s, ef) {
			continue
		}
		if ef.AlwaysOn {
			if ef.OncePer == "battle" && s.UsedBattle[ef.ID] {
				continue
			}
			running = append(running, runningItem{ef: ef, always: true})
			continue
		}
		if a := findActive(s.Active, ef.ID); a != nil {
			running = append(running, runningItem{ef: ef, opt: a.Opt})
			continue
		}
		if ef.OncePer == "turning_point" && s.Used[ef.ID] == s.TP {
			spent = append(spent, SpentCard{ID: ef.ID, Name: ef.Name, KindLabel: e.kindLabel(ef, "")})
			continue
		}
		wins := ef.Wins()
		if !(contains(wins, "any") || contains(wins, s.Ctx)) {
			continue
		}
		if ef.Requires != nil && ef.Requires.ActiveEffect != "" && !activeIDs[ef.Requires.ActiveEffect] {
			continue
		}
		q := e.quote(s, ef, "")
		usable = append(usable, e.usableCard(s, ef, q))
	}

	// chapter tactics ride along as synthetic always-on running effects
	veterans := e.veteranNames(s)
	for _, slot := range []string{"primary", "secondary", "extra"} {
		if slot == "extra" && len(veterans) == 0 {
			continue // no Chapter Veteran on the table: the extra slot is inert
		}
		id := s.Tactics[slot]
		if id == "" {
			continue
		}
		t := e.tactic(id)
		if t == nil {
			continue
		}
		syn := &rules.Effect{
			ID: "tactic." + t.ID, Name: t.Name, Kind: "chapter_tactic",
			Duration: "battle", Text: t.Text, RemindAt: t.RemindAt,
			Prompts: map[string]string{"*": t.Prompt},
		}
		running = append(running, runningItem{ef: syn, always: true, slot: slot})
	}

	for i := range running {
		running[i].now = fires(running[i], triggers)
	}

	// prompts: surfacing running items not dismissed this window
	prompts := []Prompt{}
	var fresh []string
	for _, ri := range running {
		if !ri.now {
			continue
		}
		var opt *rules.Option
		if ri.opt != "" {
			for j := range ri.ef.Options {
				if ri.ef.Options[j].ID == ri.opt {
					opt = &ri.ef.Options[j]
					break
				}
			}
		}
		key := ri.ef.ID + "|" + strconv.Itoa(s.TP) + "|" + s.Ctx
		if s.Dismissed[key] {
			continue
		}
		name := ri.ef.Name
		text := ri.ef.Text
		if opt != nil {
			name += " · " + opt.Name
			text = opt.Prompt
		} else if len(ri.ef.Prompts) > 0 {
			text = promptText(ri.ef, triggers)
		}
		paid := !ri.always
		prompts = append(prompts, Prompt{ID: ri.ef.ID, Name: name, Text: text, Key: key, Paid: paid})
		if paid && !s.Surfaced[ri.ef.ID] {
			fresh = append(fresh, ri.ef.ID)
		}
	}

	// usable: affordable first, then by kind
	sort.SliceStable(usable, func(i, j int) bool {
		if usable[i].Afford != usable[j].Afford {
			return usable[i].Afford
		}
		return usable[i].Kind < usable[j].Kind
	})

	// build running cards, now-first
	sort.SliceStable(running, func(i, j int) bool { return running[i].now && !running[j].now })
	runningCards := make([]RunningCard, 0, len(running))
	for _, ri := range running {
		runningCards = append(runningCards, e.runningCard(ri))
	}

	flagged := 0
	for i := range e.Rules.Effects {
		if len(e.Rules.Effects[i].Verify) > 0 {
			flagged++
		}
	}

	leaders := 0
	for _, x := range s.Roster {
		if e.isLeader(x) {
			leaders++
		}
	}

	return &View{
		// Team is filled in by the store, which knows the game's route key.
		TP: s.TP, CP: s.CP, Ctx: s.Ctx,
		Roster: s.Roster, Op: s.Op, Dead: s.Dead, Equip: s.Equip, Tactics: s.Tactics,
		CanUndo:   logLen > 0,
		CanPrevTP: s.TP > 1,
		Roster6:   RosterStatus{Total: len(s.Roster), Leaders: leaders, OK: len(s.Roster) == 6 && leaders == 1},
		Veterans:  veterans,
		Prompts:   prompts,
		Summary:   s.LastSummary,
		Usable:    usable,
		Running:   runningCards,
		Spent:     spent,
		VerifyNote: VerifyNote{
			Flagged:    flagged,
			Unresolved: e.Rules.Meta.Unresolved,
		},
		freshPrompt: fresh,
	}
}

func (e *Engine) usableCard(s *State, ef *rules.Effect, q quoted) UsableCard {
	base := ef.Cost.CP
	reduced := q.CP < base
	price := strconv.Itoa(q.CP) + " CP"
	if q.CP == 0 {
		price = "Free"
	}
	shown := price
	if reduced {
		shown = strconv.Itoa(base) + " CP → " + price
	}
	c := UsableCard{
		ID: ef.ID, Name: ef.Name, Kind: ef.Kind, KindLabel: e.kindLabel(ef, ""),
		Text: ef.Text, CostBase: base, CostShown: shown, Reduced: reduced,
		Free: q.CP == 0, Afford: s.CP >= q.CP, Condition: q.Condition, From: q.From,
	}
	for _, r := range q.Maybe {
		c.Maybe = append(c.Maybe, MaybeRoute{
			From: r.From, Options: r.Options, Condition: r.Condition,
			Needs: r.Needs != "", Disputed: r.Disputed,
		})
	}
	for j := range ef.Options {
		o := &ef.Options[j]
		oq := e.quote(s, ef, o.ID)
		c.Options = append(c.Options, OptionQuote{
			ID: o.ID, Name: o.Name, Condition: o.Condition, CP: oq.CP, Free: oq.CP == 0,
		})
	}
	return c
}

func (e *Engine) runningCard(ri runningItem) RunningCard {
	ef := ri.ef
	var meta string
	if ri.always {
		if ef.OncePer == "battle" {
			meta = "once per battle · not yet used"
		} else {
			meta = "always on"
		}
	} else if lbl, ok := expiryLabels[ef.Duration]; ok {
		meta = lbl
	} else {
		meta = ef.Duration
	}
	text := ef.Text
	if ri.opt != "" {
		for j := range ef.Options {
			if ef.Options[j].ID == ri.opt {
				text = ef.Options[j].Prompt
				break
			}
		}
	}
	return RunningCard{
		ID: ef.ID, Name: ef.Name, KindLabel: e.kindLabel(ef, ri.slot), Text: text,
		Meta: meta, Now: ri.now, Always: ri.always,
		CanEnd:      !ri.always,
		CanMarkUsed: ri.always && ef.OncePer == "battle",
		Disputed:    ef.Disputed,
	}
}

func (e *Engine) veteranNames(s *State) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, inst := range s.Roster {
		o := e.opFor(inst)
		if o != nil && o.ChapterVeteran && !seen[o.ID] {
			seen[o.ID] = true
			out = append(out, o.Name)
		}
	}
	return out
}

func (e *Engine) tactic(id string) *rules.ChapterTactic {
	for i := range e.Rules.ChapterTactics {
		if e.Rules.ChapterTactics[i].ID == id {
			return &e.Rules.ChapterTactics[i]
		}
	}
	return nil
}

func findActive(active []ActiveEntry, id string) *ActiveEntry {
	for i := range active {
		if active[i].ID == id {
			return &active[i]
		}
	}
	return nil
}

func promptText(ef *rules.Effect, triggers []string) string {
	for _, t := range triggers {
		if v, ok := ef.Prompts[t]; ok {
			return v
		}
	}
	if v, ok := ef.Prompts["*"]; ok {
		return v
	}
	return ef.Text
}
