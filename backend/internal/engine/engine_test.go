package engine

import (
	"path/filepath"
	"testing"

	"killteam/internal/rules"
)

// loadTeam loads a team's census for tests.
func loadTeam(t *testing.T, id string) *Engine {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "data", "teams")
	d, err := rules.Load(dir, id)
	if err != nil {
		t.Fatalf("load rules %s: %v", id, err)
	}
	return New(d)
}

// loadEngine loads the canonical AoD census for tests.
func loadEngine(t *testing.T) *Engine { return loadTeam(t, "aod") }

func ev(t string, p Params) Event { return Event{T: t, P: p} }

// find returns the first card matching id, or nil.
func findUsable(cards []UsableCard, id string) *UsableCard {
	for i := range cards {
		if cards[i].ID == id {
			return &cards[i]
		}
	}
	return nil
}
func findRunning(cards []RunningCard, id string) *RunningCard {
	for i := range cards {
		if cards[i].ID == id {
			return &cards[i]
		}
	}
	return nil
}

func TestBlankState(t *testing.T) {
	e := loadEngine(t)
	s := e.Fold(nil)
	if s.TP != 1 || s.CP != 3 || s.Ctx != "strategy_phase" {
		t.Fatalf("blank got tp=%d cp=%d ctx=%q", s.TP, s.CP, s.Ctx)
	}
	if len(s.Roster) != 6 || s.Op != "captain" {
		t.Fatalf("blank roster=%v op=%q", s.Roster, s.Op)
	}
}

func TestCombatDoctrineActivateThenRuns(t *testing.T) {
	e := loadEngine(t)
	log := []Event{
		ev(EvActivate, Params{ID: "aod.strat.combat_doctrine", Opt: "devastator"}),
	}
	s := e.Fold(log)
	if s.CP != 2 {
		t.Fatalf("expected 2 CP (3 at TP1, minus 1) after paying for Combat Doctrine, got %d", s.CP)
	}
	// In strategy phase it should be running, not usable.
	v := e.Derive(s, len(log))
	if findUsable(v.Usable, "aod.strat.combat_doctrine") != nil {
		t.Fatal("Combat Doctrine should not be usable once active")
	}
	if findRunning(v.Running, "aod.strat.combat_doctrine") == nil {
		t.Fatal("Combat Doctrine should be running")
	}

	// Switch to shooting: the Devastator prompt should fire.
	log = append(log, ev(EvCtx, Params{Ctx: "combat.shoot"}))
	s = e.Fold(log)
	v = e.Derive(s, len(log))
	rc := findRunning(v.Running, "aod.strat.combat_doctrine")
	if rc == nil || !rc.Now {
		t.Fatalf("Devastator should apply now in combat.shoot: %+v", rc)
	}
	if len(v.Prompts) == 0 {
		t.Fatal("expected a surfacing prompt in combat.shoot")
	}
	if len(v.FreshSurfaces()) == 0 {
		t.Fatal("expected a fresh paid surface to record")
	}
}

func TestHeroicLeaderDiscountsFirefightPloys(t *testing.T) {
	e := loadEngine(t)
	s := e.Fold(nil) // default op is the Captain
	shock := e.effect("aod.ff.shock_assault")
	q := e.quote(s, shock, "")
	if q.CP != 0 || q.From != "Heroic Leader" {
		t.Fatalf("Heroic Leader should make Shock Assault free: cp=%d from=%q", q.CP, q.From)
	}
	// Command Re-roll is explicitly excluded.
	reroll := e.effect("core.ff.command_reroll")
	if q2 := e.quote(s, reroll, ""); q2.CP != 1 {
		t.Fatalf("Command Re-roll must not be discounted: cp=%d", q2.CP)
	}
}

func TestHeroicLeaderNeedsCaptainSelected(t *testing.T) {
	e := loadEngine(t)
	// Select a non-Captain operative: the discount should become a "maybe".
	s := e.Fold([]Event{ev(EvOp, Params{ID: "intercessor_gunner"})})
	q := e.quote(s, e.effect("aod.ff.shock_assault"), "")
	if q.CP != 1 {
		t.Fatalf("discount should not apply headline when Captain not selected: cp=%d", q.CP)
	}
	if len(q.Maybe) == 0 {
		t.Fatal("expected a dim 'can be free' route when Captain not selected")
	}
}

func TestDoctrineWarfareFreeForSergeant(t *testing.T) {
	e := loadEngine(t)
	s := e.Fold([]Event{ev(EvLeader, Params{ID: "intercessor_sergeant"})})
	cd := e.effect("aod.strat.combat_doctrine")
	if q := e.quote(s, cd, "devastator"); q.CP != 0 {
		t.Fatalf("Intercessor Sergeant should get Devastator free: cp=%d", q.CP)
	}
	if q := e.quote(s, cd, "tactical"); q.CP != 0 {
		t.Fatalf("Intercessor Sergeant should get Tactical free: cp=%d", q.CP)
	}
	// Assault is not in the Intercessor Sergeant's free options.
	if q := e.quote(s, cd, "assault"); q.CP != 1 {
		t.Fatalf("Assault should cost 1 for the Intercessor Sergeant: cp=%d", q.CP)
	}
}

func TestOncePerTurningPointGoesToSpent(t *testing.T) {
	e := loadTeam(t, "plague_marines")
	// Core rules: every ploy except Command Re-roll is once per turning point.
	// Curse of Rot is instant, so after use it should sit in "spent".
	log := []Event{
		ev(EvCtx, Params{Ctx: "combat.shoot"}),
		ev(EvActivate, Params{ID: "pm.ff.curse_of_rot"}),
	}
	s := e.Fold(log)
	v := e.Derive(s, len(log))
	if findUsable(v.Usable, "pm.ff.curse_of_rot") != nil {
		t.Fatal("Curse of Rot should not be usable twice in one turning point")
	}
	found := false
	for _, sp := range v.Spent {
		if sp.ID == "pm.ff.curse_of_rot" {
			found = true
		}
	}
	if !found {
		t.Fatal("Curse of Rot should appear in spent")
	}
	// Command Re-roll is the one exception: still usable after use.
	log = append(log, ev(EvActivate, Params{ID: "core.ff.command_reroll"}))
	s = e.Fold(log)
	if findUsable(e.Derive(s, len(log)).Usable, "core.ff.command_reroll") == nil {
		t.Fatal("Command Re-roll has no per-turning-point limit")
	}
}

func TestEveryPloyExceptRerollIsOncePerTurningPoint(t *testing.T) {
	for _, team := range []string{"aod", "plague_marines"} {
		e := loadTeam(t, team)
		for _, ef := range e.Rules.Effects {
			isPloy := ef.Kind == "strategy_ploy" || ef.Kind == "firefight_ploy"
			if !isPloy {
				continue
			}
			if ef.ID == "core.ff.command_reroll" {
				if ef.OncePer != "" {
					t.Errorf("%s: Command Re-roll must be unlimited", team)
				}
				continue
			}
			if ef.OncePer != "turning_point" {
				t.Errorf("%s: %s must be once per turning point, got %q", team, ef.ID, ef.OncePer)
			}
			if ef.Cost.CP != 1 {
				t.Errorf("%s: %s must cost 1CP (core rules), got %d", team, ef.ID, ef.Cost.CP)
			}
		}
	}
}

func TestCommandPointIncomeFollowsInitiative(t *testing.T) {
	e := loadEngine(t)
	s := e.Fold([]Event{ev(EvTPNext, Params{Ini: "us"})})
	if s.CP != 4 {
		t.Fatalf("with initiative: 3 + 1 = 4 CP, got %d", s.CP)
	}
	s = e.Fold([]Event{ev(EvTPNext, Params{Ini: "them"})})
	if s.CP != 5 {
		t.Fatalf("without initiative: 3 + 2 = 5 CP, got %d", s.CP)
	}
}

func TestTurningPointSummaryFlagsUnusedEffects(t *testing.T) {
	e := loadEngine(t)
	// Activate an end-of-turning-point effect, then advance without surfacing it.
	log := []Event{
		ev(EvActivate, Params{ID: "aod.strat.combat_doctrine", Opt: "devastator"}),
		ev(EvTPNext, Params{}),
	}
	s := e.Fold(log)
	if s.TP != 2 {
		t.Fatalf("expected TP 2, got %d", s.TP)
	}
	if s.LastSummary == nil || len(s.LastSummary.Names) == 0 {
		t.Fatal("expected an end-of-turning-point 'never used' summary")
	}
	// Advancing should also clear the active list and bump CP.
	if len(s.Active) != 0 {
		t.Fatalf("active list should clear on TP_NEXT, got %v", s.Active)
	}
}

func TestSurfacedEffectNotFlagged(t *testing.T) {
	e := loadEngine(t)
	log := []Event{
		ev(EvActivate, Params{ID: "aod.strat.combat_doctrine", Opt: "devastator"}),
		ev(EvSurface, Params{ID: "aod.strat.combat_doctrine"}), // as the store would append
		ev(EvTPNext, Params{}),
	}
	s := e.Fold(log)
	if s.LastSummary != nil {
		t.Fatalf("a surfaced effect must not be flagged as unused: %+v", s.LastSummary)
	}
}

func TestEquipmentPresenceGate(t *testing.T) {
	e := loadEngine(t)
	// Auspex is equipment; without taking it, it should not be usable in shooting.
	log := []Event{ev(EvCtx, Params{Ctx: "combat.shoot"})}
	s := e.Fold(log)
	if findUsable(e.Derive(s, len(log)).Usable, "aod.eq.auspex") != nil {
		t.Fatal("Auspex should be absent until equipped")
	}
	log = append(log, ev(EvEquip, Params{ID: "aod.eq.auspex"}))
	s = e.Fold(log)
	if findUsable(e.Derive(s, len(log)).Usable, "aod.eq.auspex") == nil {
		t.Fatal("Auspex should be usable once equipped, in the shooting window")
	}
}

func TestPlagueMarinesDefaultRosterFromMeta(t *testing.T) {
	e := loadTeam(t, "plague_marines")
	s := e.Fold(nil)
	if len(s.Roster) != 6 {
		t.Fatalf("expected 6 operatives, got %d: %v", len(s.Roster), s.Roster)
	}
	if s.Op != "plague_marine_champion" {
		t.Fatalf("expected the Champion to start selected, got %q", s.Op)
	}
	// The engine must not be hardcoded to AoD ids.
	for _, inst := range s.Roster {
		if typeOf(inst) == "captain" {
			t.Fatal("Plague Marines roster leaked an Angels of Death operative")
		}
	}
}

func TestIconOfContagionDiscount(t *testing.T) {
	e := loadTeam(t, "plague_marines")
	// Icon Bearer is in the default roster: Contagion should quote free, with the
	// board condition surfaced (the app can't verify "opponent's territory").
	s := e.Fold(nil)
	contagion := e.effect("pm.strat.contagion")
	if q := e.quote(s, contagion, ""); q.CP != 0 || q.From != "Icon of Contagion" {
		t.Fatalf("Contagion should be free via Icon of Contagion: cp=%d from=%q", q.CP, q.From)
	}
	// Remove the Icon Bearer: the discount disappears.
	s = e.Fold([]Event{ev(EvRoster, Params{ID: "plague_marine_icon_bearer"})})
	if q := e.quote(s, contagion, ""); q.CP != 1 {
		t.Fatalf("without the Icon Bearer, Contagion should cost 1: cp=%d", q.CP)
	}
}

func TestContextClearsShortDurations(t *testing.T) {
	e := loadEngine(t)
	// Shock Assault is this_activation; it should drop when the context changes.
	log := []Event{
		ev(EvCtx, Params{Ctx: "combat.fight"}),
		ev(EvActivate, Params{ID: "aod.ff.shock_assault"}),
	}
	s := e.Fold(log)
	if findActive(s.Active, "aod.ff.shock_assault") == nil {
		t.Fatal("Shock Assault should be active right after use")
	}
	log = append(log, ev(EvCtx, Params{Ctx: "strategy_phase"}))
	s = e.Fold(log)
	if findActive(s.Active, "aod.ff.shock_assault") != nil {
		t.Fatal("Shock Assault (this_activation) should clear when context changes")
	}
}
