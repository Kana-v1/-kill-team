import Foundation

/// One game: its team and its append-only event log. This is what the app
/// persists; everything shown is derived from it.
public struct GameLog: Codable, Equatable {
    public var id: String
    public var team: String
    public var events: [Event]

    public init(id: String = UUID().uuidString, team: String, events: [Event] = []) {
        self.id = id
        self.team = team
        self.events = events
    }

    public mutating func append(_ e: Event) { events.append(e) }

    /// Undo is dropping the last event; state is re-folded from what's left.
    public mutating func undo() {
        if !events.isEmpty { events.removeLast() }
    }

    public mutating func reset() { events = [] }

    public var canUndo: Bool { !events.isEmpty }
}
