import SwiftUI
import UniformTypeIdentifiers
import KTEngine

/// Settings-style setup (canvas "Setup"): kill team, leader, operatives,
/// chapter tactics, equipment, and which official rules document is loaded.
struct SetupView: View {
    @EnvironmentObject private var store: GameStore
    @Environment(\.dismiss) private var dismiss
    @State private var pendingTeam: TeamInfo?
    @State private var confirmReset = false
    @State private var importing = false
    @State private var importResult: String?

    private var snap: Snapshot { store.snapshot }
    private var rules: RulesData { store.rules }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    ForEach(store.teams) { t in
                        Button {
                            if t.id != store.game.team { pendingTeam = t }
                        } label: {
                            HStack {
                                Circle().fill(Theme.accent(t.id)).frame(width: 10, height: 10)
                                Text(t.name).foregroundStyle(Theme.text)
                                Spacer()
                                if t.id == store.game.team { Image(systemName: "checkmark").foregroundStyle(Theme.link) }
                            }
                        }
                    }
                } header: { Text("Kill team") } footer: { Text("Switching kill team starts a new game.") }

                Section {
                    ForEach(rules.operatives.filter(\.leader), id: \.id) { o in
                        Button { store.send(Event(.leader, Params(id: o.id))) } label: {
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(o.name).foregroundStyle(Theme.text)
                                    Text(grants(o)).font(.caption).foregroundStyle(Theme.text2)
                                }
                                Spacer()
                                if count(o.id) > 0 { Image(systemName: "checkmark").foregroundStyle(Theme.link) }
                            }
                        }
                    }
                } header: { Text("Leader") } footer: { Text("Your leader changes which abilities and discounts you get.") }

                Section {
                    ForEach(rules.operatives.filter { !$0.leader }, id: \.id) { o in
                        if o.multiple {
                            Stepper(value: Binding(get: { count(o.id) }, set: { new in
                                store.send(Event(.count, Params(d: new > count(o.id) ? 1 : -1, id: o.id)))
                            }), in: 0...6) {
                                Text("\(o.name) · \(count(o.id))")
                            }
                        } else {
                            Toggle(o.name, isOn: Binding(get: { count(o.id) > 0 },
                                                         set: { _ in store.send(Event(.roster, Params(id: o.id))) }))
                        }
                    }
                } header: {
                    HStack {
                        Text("Operatives")
                        Spacer()
                        Text(rosterLabel).foregroundStyle(snap.rosterStatus.ok ? Theme.go : Theme.hot)
                    }
                }

                if !rules.chapterTactics.isEmpty {
                    Section("Chapter tactics") {
                        tacticLink("Primary", slot: "primary")
                        tacticLink("Secondary", slot: "secondary")
                        if !snap.veterans.isEmpty {
                            tacticLink("Extra — \(snap.veterans.joined(separator: " / ")) only", slot: "extra")
                        }
                    }
                }

                Section {
                    ForEach(rules.effects.filter { $0.kind == "equipment" }, id: \.id) { e in
                        HStack {
                            InfoLabel(id: e.id, name: e.name)
                            Spacer()
                            Toggle(e.name, isOn: Binding(get: { snap.equip.contains(e.id) },
                                                         set: { _ in store.send(Event(.equip, Params(id: e.id))) }))
                                .labelsHidden()
                        }
                    }
                } header: { Text("Faction equipment") } footer: { Text("Tap a name to read what it does.") }

                Section {
                    LabeledContent("Have photos", value: "\(photoCount) of \(rules.operatives.count)")
                    Button("Import photos…") { importing = true }
                    if let importResult { Text(importResult).font(.footnote).foregroundStyle(Theme.text2) }
                } header: { Text("Operative photos") } footer: {
                    Text("Name each file after its operative (e.g. plague_marine_champion.png). You can also drop them into Kill Team's folder in the Files app; they're picked up when the app opens. To set one photo, tap the portrait on an operative.")
                }

                Section {
                    LabeledContent("Rules version", value: rules.meta.rulesVersion ?? "unknown")
                } header: { Text("Rules") } footer: {
                    Text("From the latest official rules. Anything not yet confirmed is tagged Unverified in the game.")
                }

                Section {
                    Button("Reset whole game", role: .destructive) { confirmReset = true }
                }
            }
            .scrollContentBackground(.hidden)
            .background(Theme.bg)
            .navigationTitle("Setup")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) { Button("Done") { dismiss() } }
            }
            .confirmationDialog("Start a new \(pendingTeam?.name ?? "") game?",
                                isPresented: Binding(get: { pendingTeam != nil }, set: { if !$0 { pendingTeam = nil } }),
                                titleVisibility: .visible) {
                Button("New game") {
                    if let t = pendingTeam { store.switchTeam(t.id) }
                    pendingTeam = nil
                }
            } message: { Text("The current game will be replaced.") }
            .fileImporter(isPresented: $importing, allowedContentTypes: [.image], allowsMultipleSelection: true) { result in
                guard case .success(let urls) = result else { return }
                let r = store.importPhotos(from: urls)
                importResult = "Added \(r.matched)." + (r.unmatched.isEmpty ? "" : " Didn't match: " + r.unmatched.joined(separator: ", "))
            }
            .ruleInfo(store.engine)
            .confirmationDialog("Reset the whole game?", isPresented: $confirmReset, titleVisibility: .visible) {
                Button("Reset", role: .destructive) { store.resetGame() }
            }
        }
        .preferredColorScheme(.dark)
    }

    private func count(_ id: String) -> Int { snap.roster.filter { typeOf($0) == id }.count }

    private var rosterLabel: String {
        "\(snap.rosterStatus.total) of 6" + (snap.rosterStatus.leaders == 1 ? "" : " · no leader")
    }

    private var photoCount: Int { rules.operatives.filter { store.photo(for: $0.id) != nil }.count }

    /// Abilities a leader brings, beyond the faction rule everyone has.
    private func grants(_ o: Operative) -> String {
        let own = o.abilities.filter { a in !rules.effects.contains { $0.kind == "faction_rule" && $0.name == a } }
        return own.isEmpty ? "No extra abilities" : own.joined(separator: " · ")
    }

    private func tacticLink(_ title: String, slot: String) -> some View {
        NavigationLink {
            TacticChooser(slot: slot, title: title).environmentObject(store)
        } label: {
            LabeledContent(title, value: rules.chapterTactics.first { $0.id == snap.tactics[slot] }?.name ?? "None")
        }
    }
}

/// A rule name in Setup that opens its full text.
struct InfoLabel: View {
    let id: String
    let name: String
    @Environment(\.showInfo) private var showInfo

    var body: some View {
        Button { showInfo(.rule(id)) } label: {
            HStack(spacing: 6) {
                Text(name).foregroundStyle(Theme.text)
                Image(systemName: "info.circle").font(.footnote).foregroundStyle(Theme.link)
            }
        }
        .buttonStyle(.plain)
    }
}

/// Pick a chapter tactic, reading each one's full rule (terms tappable).
struct TacticChooser: View {
    @EnvironmentObject private var store: GameStore
    @Environment(\.dismiss) private var dismiss
    let slot: String
    let title: String

    var body: some View {
        List {
            ForEach(store.rules.chapterTactics, id: \.id) { t in
                let chosen = store.snapshot.tactics[slot] == t.id
                HStack(alignment: .top, spacing: 12) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(t.name).font(.headline)
                        RuleText(segments: store.engine.segments(t.text, excluding: t.name), font: .footnote)
                    }
                    Spacer()
                    Button {
                        store.send(Event(.tactic, Params(id: t.id, slot: slot)))
                        if !chosen { dismiss() }
                    } label: {
                        Image(systemName: chosen ? "checkmark.circle.fill" : "circle")
                            .font(.title3).foregroundStyle(chosen ? Theme.link : Theme.muted)
                            .frame(width: 44, height: 44)
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(chosen ? "Clear \(t.name)" : "Choose \(t.name)")
                }
            }
        }
        .navigationTitle(title)
        .scrollContentBackground(.hidden)
        .background(Theme.bg)
        .ruleInfo(store.engine)
    }
}
