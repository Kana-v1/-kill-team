import Foundation

// The rules census for one kill team, decoded from data/teams/<id>.json.
// Reference data only: it says what exists, never what's happening in a game.
// Field meanings are documented in CLAUDE.md ("Data model").

public struct RulesData: Codable {
    public var meta: Meta
    public var vocab: Vocab
    public var effects: [Effect]
    public var operatives: [Operative]
    public var universalEquipment: [Weapon]?
    public var chapterTactics: [ChapterTactic]
    /// Faction terms (Poison, Toxic, …) for the tappable glossary.
    public var glossary: [String: GlossaryEntry]?

    public static func load(from url: URL) throws -> RulesData {
        try JSONDecoder().decode(RulesData.self, from: Data(contentsOf: url))
    }
}

public struct Meta: Codable {
    public var team: String
    public var teamId: String
    public var edition: String?
    public var sourcePdf: String?
    public var rulesVersion: String?
    public var defaultRoster: [String]?
    public var unresolved: [String]?
}

public struct Vocab: Codable {
    public var phases: [String]?
    public var when: [String]?
    public var appliesTo: [String]?
}

public struct Cost: Codable, Equatable {
    public var cp: Int
}

public struct Effect: Codable {
    public var id: String
    public var name: String
    public var kind: String
    public var universal: Bool?
    public var cost: Cost
    public var duration: String
    public var oncePer: String?
    public var text: String
    public var options: [Option]?
    public var requiresOperative: String?
    public var alwaysOn: Bool?
    public var costOverride: Override?
    public var costOverrides: [Override]?
    public var changesOptionOf: String?
    public var requires: Requires?
    public var disputed: Bool?
    public var verify: [String]?
    // iOS model fields (tools/add_ios_model_fields.py)
    public var when: When?
    public var appliesTo: AppliesTo?
    public var weaponMatch: [String]?
    public var hint: String?
    public var grantsWeaponRules: [Grant]?
    /// A moment the app itself sees that should offer this ploy ("incapacitated").
    public var trigger: String?

    enum CodingKeys: String, CodingKey {
        case id, name, kind, universal, cost, duration, text, options, requiresOperative, alwaysOn,
             costOverride, costOverrides, changesOptionOf, requires, disputed, verify,
             when, appliesTo, weaponMatch, hint, grantsWeaponRules, trigger
        case oncePer = "once_per"
    }

    /// Cost overrides as one list (the singular field is legacy).
    public var overrides: [Override] {
        if let list = costOverrides, !list.isEmpty { return list }
        if let one = costOverride { return [one] }
        return []
    }

    public var isAlwaysOn: Bool { alwaysOn ?? false }
    public var isPloy: Bool { kind == "strategy_ploy" || kind == "firefight_ploy" }
}

public struct Option: Codable {
    public var id: String
    public var name: String
    public var condition: String?
    public var prompt: String?
    public var hint: String?
    public var grantsWeaponRules: [Grant]?
}

public struct Override: Codable {
    public var effect: String?
    public var kind: String?
    public var options: [String]?
    public var excludes: [String]?
    public var cp: Int
    public var oncePer: String?
    public var group: String?
    public var selectedIs: String?
    public var condition: String?

    enum CodingKeys: String, CodingKey {
        case effect, kind, options, excludes, cp, group, selectedIs, condition
        case oncePer = "once_per"
    }
}

public struct Requires: Codable {
    public var note: String?
    public var activeEffect: String?
}

/// Rules an effect adds to weapons, shown as notes under the weapon row.
/// `match`: "*" (all), "ranged", "melee", or case-insensitive weapon-name substrings.
public struct Grant: Codable, Equatable {
    public var match: [String]
    public var rules: [String]
    public var condition: String?
}

public struct Operative: Codable {
    public var id: String
    public var name: String
    public var stats: Stats
    public var abilities: [String]
    public var weapons: [Weapon]
    public var icon: String
    public var accent: String
    public var photo: String?
    public var leader: Bool
    public var multiple: Bool
    public var role: String
    public var chapterVeteran: Bool
}

public struct Stats: Codable, Equatable {
    public var apl: Int
    public var move: String
    public var save: String
    public var wounds: Int
}

public struct Weapon: Codable, Equatable {
    public var name: String
    public var type: String
    public var atk: Int
    public var hit: String
    public var dmg: String
    public var rules: String

    public var isMelee: Bool { type == "melee" }
}

public struct ChapterTactic: Codable {
    public var id: String
    public var name: String
    public var text: String
    public var prompt: String?
    public var when: When?
    public var hint: String?
    public var grantsWeaponRules: [Grant]?
}

public struct GlossaryEntry: Codable, Equatable {
    public var kind: String
    public var def: String
}

/// data/core/glossary.json: core weapon rules, from the official Lite rules.
public struct CoreGlossary: Codable {
    public var terms: [String: GlossaryEntry]
    public var keywordsNotRules: [String]?

    public static func load(from url: URL) throws -> CoreGlossary {
        try JSONDecoder().decode(CoreGlossary.self, from: Data(contentsOf: url))
    }
}

/// Which group a rule shows in during the firefight.
public enum When: String, Codable, CaseIterable {
    case activation, attack, defence, any

    public var title: String {
        switch self {
        case .activation: return "Your activation"
        case .attack: return "Your attacks"
        case .defence: return "When you're attacked"
        case .any: return "Any time"
        }
    }
}

/// Who a rule applies to: every friendly operative, only its own operative,
/// or only operatives carrying a matching weapon.
public enum AppliesTo: String, Codable {
    case team, `self`, weapons
}
