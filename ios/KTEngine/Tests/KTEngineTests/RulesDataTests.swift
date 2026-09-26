import XCTest
@testable import KTEngine

/// Checks on the census files themselves — they fail when data is edited into
/// an inconsistent or rules-breaking shape.
final class RulesDataTests: XCTestCase {
    func testPloysFollowTheCoreRules() {
        // Official Lite rules: ploys cost 1CP; each except Command Re-roll once per turning point.
        for team in Repo.teams {
            for e in Repo.rules(team).effects where e.isPloy {
                XCTAssertEqual(e.cost.cp, 1, "\(team): \(e.name)")
                if e.id == "core.ff.command_reroll" {
                    XCTAssertNil(e.oncePer, "Command Re-roll has no limit")
                } else {
                    XCTAssertEqual(e.oncePer, "turning_point", "\(team): \(e.name)")
                }
            }
        }
    }

    func testModelFieldsAreCompleteAndConsistent() {
        for team in Repo.teams {
            let r = Repo.rules(team)
            let opIds = Set(r.operatives.map(\.id))
            for e in r.effects {
                XCTAssertNotNil(e.when, "\(team): \(e.name) has no `when`")
                XCTAssertNotNil(e.appliesTo, "\(team): \(e.name) has no `appliesTo`")
                XCTAssertFalse((e.hint ?? "").isEmpty, "\(team): \(e.name) has no hint")
                if let who = e.requiresOperative {
                    XCTAssertTrue(opIds.contains(who), "\(team): \(e.name) requires unknown operative \(who)")
                }
                switch e.appliesTo {
                case .self?:
                    XCTAssertNotNil(e.requiresOperative, "\(team): \(e.name) is `self` but names no operative")
                case .weapons?:
                    let m = e.weaponMatch ?? []
                    XCTAssertFalse(m.isEmpty, "\(team): \(e.name) is `weapons` but matches nothing")
                    let anyCarrier = r.operatives.contains { o in o.weapons.contains { w in m.contains { w.name.lowercased().contains($0) } } }
                    XCTAssertTrue(anyCarrier, "\(team): no operative carries a weapon matching \(m)")
                default:
                    break
                }
            }
            for t in r.chapterTactics {
                XCTAssertNotNil(t.when, "\(team): tactic \(t.name) has no `when`")
            }
        }
    }

    func testDataIsTracedToTheOfficialDocument() {
        for team in Repo.teams {
            let m = Repo.rules(team).meta
            XCTAssertNotNil(m.sourcePdf, "\(team): record which official PDF the census was checked against")
            XCTAssertNotNil(m.rulesVersion, "\(team): record the errata month")
        }
    }
}
