import Foundation
import SwiftUI
import KTEngine

struct TeamInfo: Identifiable, Hashable {
    let id: String
    let name: String
}

/// Owns the current game. Every tap is one event: append, re-fold, re-derive,
/// save. There is no server; the rules engine runs on the phone.
@MainActor
final class GameStore: ObservableObject {
    @Published private(set) var game: GameLog
    @Published private(set) var snapshot: Snapshot
    /// "Fighter is acting — Sickening Resilience ended." after a new activation.
    @Published var toast: String?

    let teams: [TeamInfo]
    private let engines: [String: Engine]

    var engine: Engine { engines[game.team]! }
    var rules: RulesData { engine.rules }
    var state: GameState { engine.fold(game.events) }

    private static let saveURL: URL = {
        let dir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
        return dir.appendingPathComponent("game.json")
    }()

    init() {
        let (engines, teams) = GameStore.loadRules()
        self.engines = engines
        self.teams = teams
        var game = GameStore.loadGame() ?? GameLog(team: teams.first?.id ?? "aod")
        if engines[game.team] == nil { game = GameLog(team: teams.first?.id ?? "aod") }
        self.game = game
        let engine = engines[game.team]!
        snapshot = engine.derive(engine.fold(game.events), logLength: game.events.count)
        Log.write("loaded \(teams.count) teams; game \(game.team) with \(game.events.count) events", "store")
    }

    // MARK: actions

    func send(_ event: Event) {
        let before = state
        game.append(event)
        refresh()
        if event.t == .op, let id = event.p.id, id != before.op {
            announceNewActivation(before: before, now: id)
        }
    }

    func undo() {
        game.undo()
        toast = nil
        refresh()
    }

    func resetGame() {
        game.reset()
        refresh()
    }

    /// A game is bound to one team, so switching team starts a new game.
    func switchTeam(_ team: String) {
        guard team != game.team, engines[team] != nil else { return }
        game = GameLog(team: team)
        refresh()
    }

    private func announceNewActivation(before: GameState, now id: String) {
        let ended = before.active.filter { a in !state.active.contains(where: { $0.id == a.id }) }
            .compactMap { engine.effect($0.id)?.name }
        guard !ended.isEmpty, let op = engine.operative(id) else { return }
        toast = "\(op.name) is acting — \(ended.joined(separator: ", ")) ended."
    }

    private func refresh() {
        snapshot = engine.derive(state, logLength: game.events.count)
        save()
    }

    // MARK: persistence (best-effort: never crash over a file)

    private func save() {
        do {
            let enc = JSONEncoder()
            enc.outputFormatting = [.prettyPrinted, .sortedKeys]
            try enc.encode(game).write(to: GameStore.saveURL, options: .atomic)
        } catch {
            Log.write("save failed: \(error)", "store")
        }
    }

    private static func loadGame() -> GameLog? {
        guard let data = try? Data(contentsOf: saveURL) else { return nil }
        do {
            return try JSONDecoder().decode(GameLog.self, from: data)
        } catch {
            Log.write("saved game unreadable, starting fresh: \(error)", "store")
            return nil
        }
    }

    /// The censuses ship in the app bundle: teams/<id>.json and core/glossary.json.
    private static func loadRules() -> ([String: Engine], [TeamInfo]) {
        guard let coreURL = Bundle.main.url(forResource: "glossary", withExtension: "json", subdirectory: "core"),
              let core = try? CoreGlossary.load(from: coreURL) else {
            fatalError("core/glossary.json missing from the app bundle")
        }
        let urls = Bundle.main.urls(forResourcesWithExtension: "json", subdirectory: "teams") ?? []
        var engines: [String: Engine] = [:]
        var teams: [TeamInfo] = []
        for url in urls {
            let id = url.deletingPathExtension().lastPathComponent
            do {
                let rules = try RulesData.load(from: url)
                engines[id] = Engine(teamId: id, rules: rules, core: core)
                teams.append(TeamInfo(id: id, name: rules.meta.team))
            } catch {
                Log.write("could not load \(id).json: \(error)", "store")
            }
        }
        precondition(!engines.isEmpty, "no team censuses in the app bundle")
        return (engines, teams.sorted { $0.name < $1.name })
    }
}
