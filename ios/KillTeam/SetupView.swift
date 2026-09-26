import SwiftUI
import KTEngine

/// Settings-style setup (canvas "Setup"): kill team, leader, operatives,
/// chapter tactics, equipment, and which official rules document is loaded.
struct SetupView: View {
    @EnvironmentObject private var store: GameStore
    @Environment(\.dismiss) private var dismiss
    @State private var pendingTeam: TeamInfo?
    @State private var confirmReset = false

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

                Section("Leader") {
                    ForEach(rules.operatives.filter(\.leader), id: \.id) { o in
                        Button { store.send(Event(.leader, Params(id: o.id))) } label: {
                            HStack {
                                Text(o.name).foregroundStyle(Theme.text)
                                Spacer()
                                if count(o.id) > 0 { Image(systemName: "checkmark").foregroundStyle(Theme.link) }
                            }
                        }
                    }
                }

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
                        tacticPicker("Primary", slot: "primary")
                        tacticPicker("Secondary", slot: "secondary")
                        if !snap.veterans.isEmpty {
                            tacticPicker("Extra — \(snap.veterans.joined(separator: " / ")) only", slot: "extra")
                        }
                    }
                }

                Section("Faction equipment") {
                    ForEach(rules.effects.filter { $0.kind == "equipment" }, id: \.id) { e in
                        Toggle(e.name, isOn: Binding(get: { snap.equip.contains(e.id) },
                                                     set: { _ in store.send(Event(.equip, Params(id: e.id))) }))
                    }
                }

                Section {
                    LabeledContent("Rules version", value: rules.meta.rulesVersion ?? "unknown")
                    if let pdf = rules.meta.sourcePdf {
                        Text(pdf).font(.caption).foregroundStyle(Theme.text2)
                    }
                } header: { Text("Rules") } footer: {
                    Text("From the latest official rules PDF. Anything not yet confirmed is tagged Unverified in the game.")
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

    private func tacticPicker(_ title: String, slot: String) -> some View {
        Picker(title, selection: Binding(get: { snap.tactics[slot] ?? "" }, set: { new in
            let current = snap.tactics[slot] ?? ""
            if new.isEmpty {
                if !current.isEmpty { store.send(Event(.tactic, Params(id: current, slot: slot))) }
            } else {
                store.send(Event(.tactic, Params(id: new, slot: slot)))
            }
        })) {
            Text("None").tag("")
            ForEach(rules.chapterTactics, id: \.id) { t in Text(t.name).tag(t.id) }
        }
    }
}
