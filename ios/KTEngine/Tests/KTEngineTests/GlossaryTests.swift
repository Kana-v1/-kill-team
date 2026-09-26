import XCTest
@testable import KTEngine

final class GlossaryTests: XCTestCase {
    /// The name part of a weapon-rule token: "Lethal 5+" → matched by "Lethal",
    /// "Heavy (Dash only)" → "Heavy", "Poison*" → "Poison".
    func resolves(_ token: String, _ g: Glossary, _ keywords: [String]) -> Bool {
        let t = token.trimmingCharacters(in: .whitespaces).trimmingCharacters(in: CharacterSet(charactersIn: "*"))
        if ["", "-", "—", "Melee"].contains(t) || keywords.contains(t) { return true }
        return g.entries.keys.contains { key in
            t == key || t.hasPrefix(key + " ") || t.hasPrefix(key + "(") || t.hasPrefix(key + "s")
        }
    }

    func testEveryWeaponRuleHasADefinition() {
        let keywords = Repo.core.keywordsNotRules ?? []
        for team in Repo.teams {
            let e = Repo.engine(team)
            var missing: [String] = []
            for op in e.rules.operatives {
                for w in op.weapons {
                    for token in w.rules.split(separator: ",") where !resolves(String(token), e.glossary, keywords) {
                        missing.append("\(op.name) · \(w.name): \(token)")
                    }
                }
            }
            let grants = e.rules.effects.flatMap { ($0.grantsWeaponRules ?? []) + ($0.options ?? []).flatMap { $0.grantsWeaponRules ?? [] } }
                + e.rules.chapterTactics.flatMap { $0.grantsWeaponRules ?? [] }
            for g in grants {
                for r in g.rules where !resolves(r, e.glossary, keywords) { missing.append("granted rule: \(r)") }
            }
            XCTAssertEqual(missing, [], "\(team): weapon rules without a glossary definition")
        }
    }

    func testDefinitionsAreFromTheOfficialRulesNotParaphrase() {
        let e = Repo.engine("aod")
        // Ceaseless re-rolls one chosen result; "re-roll any" is Relentless.
        XCTAssertEqual(e.definition("Ceaseless")?.def, "You can re-roll any of your attack dice results of one result (e.g. results of 2).")
        XCTAssertEqual(e.definition("Relentless")?.def, "You can re-roll any of your attack dice.")
    }

    func testLongestTermWinsAndPluralsLink() {
        let e = Repo.engine("plague_marines")
        let segs = e.segments("Piercing Crits 1 and two Poison tokens, Poison too")
        XCTAssertEqual(segs.filter { $0.term != nil }.map(\.term), ["Piercing Crits", "Poison token", "Poison"])
        XCTAssertEqual(segs.first { $0.term == "Poison token" }?.t, "Poison tokens")
    }

    func testWholeWordsOnly() {
        let e = Repo.engine("plague_marines")
        XCTAssertTrue(e.segments("Poisonous Demise").allSatisfy { $0.term == nil }, "Poison inside Poisonous must not link")
    }

    func testAnItemNeverLinksToItself() {
        let e = Repo.engine("plague_marines")
        let segs = e.segments("Poison weapon deals damage.", excluding: "Poison")
        XCTAssertTrue(segs.allSatisfy { $0.term == nil })
    }

    func testFactionRulesAreTappableInsideOtherRules() {
        let e = Repo.engine("plague_marines")
        let v = e.derive(e.fold([ev(.phase, Params(phase: .firefight))]))
        let sickening = v.useCard("pm.ff.sickening_resilience")!
        XCTAssertEqual(sickening.hint.first { $0.term != nil }?.term, "Disgustingly Resilient")
    }

    func testActiveCardsLinkTheRulesTheyGrant() {
        let e = Repo.engine("plague_marines")
        let v = e.derive(e.fold([ev(.activate, Params(id: "pm.strat.lumbering_death"))]))
        let card = v.active.flatMap(\.cards).first { $0.id == "pm.strat.lumbering_death" }!
        XCTAssertTrue(card.hint.contains { $0.term == "Ceaseless" })
    }
}
