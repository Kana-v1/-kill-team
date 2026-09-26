import SwiftUI
import KTEngine

/// The acting operative's datacard (canvas "Operative sheet"): switch who's
/// acting, stats, every weapon with the rules effects add to it, and which of
/// your rules apply to this operative.
struct OperativeSheet: View {
    @EnvironmentObject private var store: GameStore
    @Environment(\.dismiss) private var dismiss

    private var snap: Snapshot { store.snapshot }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    rosterChips
                    if let op = store.engine.operative(snap.op) {
                        identity(op)
                        stats(op)
                        weapons(op)
                        applies
                        Button {
                            store.send(Event(.down, Params(id: snap.op)))
                        } label: {
                            Text(snap.dead.contains(snap.op) ? "Back in action" : "Mark incapacitated")
                                .font(.headline).frame(maxWidth: .infinity).frame(height: 50)
                                .background(Theme.raised, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
                                .foregroundStyle(snap.dead.contains(snap.op) ? Theme.link : Theme.hot)
                        }
                    }
                }
                .padding(16)
            }
            .background(Theme.surface)
            .navigationTitle("Operative")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) { Button("Done") { dismiss() } }
            }
            .ruleTerms(store.engine)
        }
        .preferredColorScheme(.dark)
    }

    private var rosterChips: some View {
        FlowLayout(spacing: 8) {
            ForEach(snap.roster, id: \.self) { inst in
                let selected = inst == snap.op
                let down = snap.dead.contains(inst)
                Button {
                    store.send(Event(.op, Params(id: inst)))
                } label: {
                    Text(shortName(inst))
                        .font(.subheadline.weight(.semibold))
                        .strikethrough(down)
                        .padding(.horizontal, 12).frame(minHeight: 34)
                        .foregroundStyle(selected ? Color.black : (down ? Theme.muted : Theme.text))
                        .background(selected ? Theme.text : Theme.raised, in: Capsule())
                }
                .accessibilityAddTraits(selected ? .isSelected : [])
            }
        }
    }

    private func identity(_ op: Operative) -> some View {
        HStack(spacing: 14) {
            Glyph(operative: op, size: 34)
                .frame(width: 64, height: 64)
                .background(Theme.raised, in: RoundedRectangle(cornerRadius: 18, style: .continuous))
                .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
            VStack(alignment: .leading, spacing: 2) {
                Text(op.name).font(.title2.bold())
                Text(op.role + (snap.dead.contains(snap.op) ? " · incapacitated" : ""))
                    .font(.subheadline).foregroundStyle(Theme.text2)
            }
        }
    }

    private func stats(_ op: Operative) -> some View {
        HStack(spacing: 8) {
            stat("\(op.stats.apl)", "APL")
            stat(op.stats.move, "MOVE")
            stat(op.stats.save, "SAVE")
            stat("\(op.stats.wounds)", "WOUNDS")
        }
    }

    private func stat(_ value: String, _ key: String) -> some View {
        VStack(spacing: 2) {
            Text(value).font(Theme.number(24))
            Text(key).font(.caption.weight(.semibold)).foregroundStyle(Theme.text2)
        }
        .frame(maxWidth: .infinity).padding(.vertical, 10)
        .background(Theme.raised, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    private func weapons(_ op: Operative) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionTitle(text: "Weapons", trailing: "tap a rule for its meaning")
            VStack(spacing: 0) {
                ForEach(Array(op.weapons.enumerated()), id: \.offset) { i, w in
                    if i > 0 { Hairline() }
                    weaponRow(w)
                }
            }
            .background(Theme.raised, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
        }
    }

    private func weaponRow(_ w: Weapon) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                Image(systemName: w.isMelee ? "hand.raised" : "scope").font(.footnote).foregroundStyle(Theme.muted)
                Text(w.name).font(.headline)
                Spacer(minLength: 4)
                statLabel("A", "\(w.atk)")
                statLabel(w.isMelee ? "WS" : "BS", w.hit)
                statLabel("D", w.dmg)
            }
            if !["-", "—", ""].contains(w.rules) {
                RuleText(segments: store.engine.segments(w.rules), font: .footnote)
            }
            ForEach(Array((snap.weaponNotes[w.name] ?? []).enumerated()), id: \.offset) { _, note in
                VStack(alignment: .leading, spacing: 2) {
                    RuleText(segments: store.engine.segments(note.rules.map { "+\($0)" }.joined(separator: " ")),
                             font: .subheadline.weight(.bold), color: Theme.accent(store.game.team))
                    Text(note.from + (note.condition.map { " — \($0)" } ?? ""))
                        .font(.footnote).foregroundStyle(Color(hex: 0xD6DDC8))
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 10).padding(.vertical, 8)
                .background(Theme.accent(store.game.team).opacity(0.12), in: RoundedRectangle(cornerRadius: 10))
            }
        }
        .padding(.horizontal, 14).padding(.vertical, 12)
    }

    private func statLabel(_ key: String, _ value: String) -> some View {
        HStack(spacing: 3) {
            Text(key).font(.caption).foregroundStyle(Theme.text2)
            Text(value).font(.subheadline.weight(.bold).monospacedDigit())
        }
    }

    private var applies: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionTitle(text: "Affects this operative")
            VStack(spacing: 0) {
                let cards = snap.active.flatMap(\.cards)
                ForEach(Array(cards.enumerated()), id: \.offset) { i, c in
                    if i > 0 { Hairline() }
                    HStack(spacing: 12) {
                        Image(systemName: "checkmark").font(.footnote.weight(.bold)).foregroundStyle(Theme.go)
                        Text(c.name).font(.body)
                        Spacer()
                        Text(c.kindLabel.components(separatedBy: " · ").first ?? "").font(.footnote).foregroundStyle(Theme.text2)
                    }
                    .padding(.horizontal, 14).padding(.vertical, 11)
                }
            }
            .background(Theme.raised, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
            if !snap.elsewhere.isEmpty {
                Text("On the table, but not for this operative").font(.footnote.weight(.semibold))
                    .foregroundStyle(Theme.text2).padding(.top, 6).padding(.horizontal, 4)
                ForEach(Array(snap.elsewhere.enumerated()), id: \.offset) { _, e in
                    (Text(e.name).foregroundColor(Theme.muted) + Text(" · \(e.who)").foregroundColor(Theme.text2))
                        .font(.subheadline).padding(.horizontal, 4)
                }
            }
        }
    }

    private func shortName(_ inst: String) -> String {
        guard let op = store.engine.operative(inst) else { return inst }
        var n = op.name
        for (long, short) in [("Assault Intercessor", "Asslt Int"), ("Heavy Intercessor", "Hvy Int"),
                              ("Intercessor", "Int"), ("Eliminator", "Elim"), ("Space Marine", "SM"),
                              ("Malignant Plaguecaster", "Plaguecaster"), ("Plague Marine ", "")] {
            n = n.replacingOccurrences(of: long, with: short)
        }
        let same = snap.roster.filter { typeOf($0) == typeOf(inst) }.count
        if same > 1, let num = inst.split(separator: "#").last { n += " \(num)" }
        return n
    }
}
