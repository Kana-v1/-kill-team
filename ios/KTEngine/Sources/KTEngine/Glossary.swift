import Foundation

/// A run of rule text; `term` is set when the run is a glossary term the
/// player can tap for its definition.
public struct Segment: Equatable {
    public var t: String
    public var term: String?

    public init(_ t: String, term: String? = nil) {
        self.t = t
        self.term = term
    }
}

/// Splits rule text into plain runs and tappable terms.
///
/// Terms come from three places: the core weapon rules (official Lite rules),
/// the team's own glossary (Poison, Toxic, …), and the names of its faction
/// rules (so "Disgustingly Resilient" inside another rule links to it).
/// Matching is case-sensitive, whole-word, longest-first ("Piercing Crits"
/// before "Piercing", "Poison token" before "Poison"), allows a plural "s",
/// and never links an item's own name inside its own text.
public struct Glossary {
    public let entries: [String: GlossaryEntry]
    let keys: [String]

    init(core: CoreGlossary, rules: RulesData) {
        var all = core.terms
        for e in rules.effects where e.kind == "faction_rule" {
            all[e.name] = GlossaryEntry(kind: "faction rule", def: e.text)
        }
        for (k, v) in rules.glossary ?? [:] { all[k] = v }
        entries = all
        keys = all.keys.sorted { $0.count != $1.count ? $0.count > $1.count : $0 < $1 }
    }

    public func definition(_ term: String) -> GlossaryEntry? { entries[term] }

    public func segments(_ text: String, excluding own: String? = nil) -> [Segment] {
        let chars = Array(text)
        var out: [Segment] = []
        var plain = ""
        var i = 0
        func isWord(_ c: Character) -> Bool { c.isLetter || c.isNumber }
        while i < chars.count {
            var matched: (key: String, len: Int)?
            if i == 0 || !isWord(chars[i - 1]) {
                for key in keys where key != own {
                    let k = Array(key)
                    guard i + k.count <= chars.count, Array(chars[i..<(i + k.count)]) == k else { continue }
                    var end = i + k.count
                    if end < chars.count, chars[end] == "s", end + 1 == chars.count || !isWord(chars[end + 1]) {
                        end += 1 // plural: "Poison tokens"
                    }
                    if end == chars.count || !isWord(chars[end]) {
                        matched = (key, end - i)
                        break
                    }
                }
            }
            if let m = matched {
                if !plain.isEmpty {
                    out.append(Segment(plain))
                    plain = ""
                }
                out.append(Segment(String(chars[i..<(i + m.len)]), term: m.key))
                i += m.len
            } else {
                plain.append(chars[i])
                i += 1
            }
        }
        if !plain.isEmpty { out.append(Segment(plain)) }
        return out
    }
}

extension Engine {
    public func segments(_ text: String, excluding own: String? = nil) -> [Segment] {
        glossary.segments(text, excluding: own)
    }

    public func definition(_ term: String) -> GlossaryEntry? { glossary.definition(term) }
}
