import SwiftUI
import KTEngine

// MARK: - tappable rule terms

private struct ShowTermKey: EnvironmentKey {
    static let defaultValue: (String) -> Void = { _ in }
}

extension EnvironmentValues {
    /// Opens a rule term's definition (set by `.ruleTerms(engine)`).
    var showTerm: (String) -> Void {
        get { self[ShowTermKey.self] }
        set { self[ShowTermKey.self] = newValue }
    }
}

struct TermRef: Identifiable {
    let id: String
}

/// Rule text whose terms (Ceaseless, Severe, Poison…) are tappable links that
/// open their definition. Plain runs keep the surrounding text style.
struct RuleText: View {
    let segments: [Segment]
    var font: Font = .subheadline
    var color: Color = Theme.text2
    @Environment(\.showTerm) private var showTerm

    var body: some View {
        Text(attributed)
            .font(font)
            .foregroundStyle(color)
            .tint(Theme.link)
            .fixedSize(horizontal: false, vertical: true)
            .environment(\.openURL, OpenURLAction { url in
                guard url.scheme == "ktterm",
                      let name = URLComponents(url: url, resolvingAgainstBaseURL: false)?
                        .queryItems?.first(where: { $0.name == "name" })?.value else { return .systemAction }
                showTerm(name)
                return .handled
            })
    }

    private var attributed: AttributedString {
        var out = AttributedString()
        for seg in segments {
            var part = AttributedString(seg.t)
            if let term = seg.term {
                var c = URLComponents()
                c.scheme = "ktterm"
                c.host = "term"
                c.queryItems = [URLQueryItem(name: "name", value: term)]
                part.link = c.url
                part.underlineStyle = Text.LineStyle(pattern: .dot, color: Theme.link)
            }
            out += part
        }
        return out
    }
}

/// Presents rule-term definitions for everything inside it.
struct RuleTerms: ViewModifier {
    let engine: Engine
    @State private var term: TermRef?

    func body(content: Content) -> some View {
        content
            .environment(\.showTerm, { term = TermRef(id: $0) })
            .sheet(item: $term) { ref in
                TermSheet(name: ref.id, entry: engine.definition(ref.id))
                    .presentationDetents([.height(280), .medium])
            }
    }
}

extension View {
    func ruleTerms(_ engine: Engine) -> some View { modifier(RuleTerms(engine: engine)) }
}

struct TermSheet: View {
    let name: String
    let entry: GlossaryEntry?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text((entry?.kind ?? "rule").uppercased())
                .font(.caption.weight(.semibold)).tracking(0.5).foregroundStyle(Theme.text2)
            Text(name).font(.title2.bold())
            Text(entry?.def ?? "No definition recorded for this term.")
                .font(.body).foregroundStyle(Theme.text)
                .fixedSize(horizontal: false, vertical: true)
            Spacer(minLength: 0)
        }
        .padding(20)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Theme.raised)
        .preferredColorScheme(.dark)
    }
}

// MARK: - rows

/// A ploy or piece of equipment you can use now. Tap to pay for it; ones with
/// options (Combat Doctrine) open their choices first.
struct UsableRow: View {
    let card: UsableCard
    let onUse: (String?) -> Void
    @State private var open = false

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // The price is the button: rule text stays outside it so its terms stay tappable.
            HStack(alignment: .center, spacing: 12) {
                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 6) {
                        Text(card.name).font(.headline).foregroundStyle(Theme.text)
                        if card.disputed { Tag(text: "UNVERIFIED", color: Theme.hot) }
                    }
                    RuleText(segments: card.hint)
                }
                Spacer(minLength: 8)
                Button {
                    if !card.options.isEmpty { open.toggle() } else { onUse(nil) }
                } label: {
                    CostPill(text: priceText, locked: !card.afford)
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .disabled(!card.afford && card.options.isEmpty)
                .accessibilityLabel("Use \(card.name), \(priceText)")
                .accessibilityHint(card.options.isEmpty ? "Pays for it" : "Shows its options")
            }

            if let discount = card.discount {
                RuleText(segments: discount, color: Theme.link)
            }
            ForEach(Array(card.maybe.enumerated()), id: \.offset) { _, m in
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Tag(text: "CAN BE FREE", color: Theme.link)
                    Text(maybeText(m)).font(.footnote).foregroundStyle(Theme.text2)
                }
            }
            if open {
                VStack(spacing: 0) {
                    ForEach(Array(card.options.enumerated()), id: \.element.id) { i, o in
                        if i > 0 { Hairline() }
                        Button {
                            open = false
                            onUse(o.id)
                        } label: {
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(o.name).font(.body.weight(.semibold)).foregroundStyle(Theme.text)
                                    Text(o.condition).font(.footnote).foregroundStyle(Theme.text2)
                                }
                                Spacer()
                                Text(o.cp == 0 ? "Free" : "\(o.cp) CP")
                                    .font(.system(size: 15, weight: .bold, design: .rounded))
                                    .foregroundStyle(Theme.go)
                            }
                            .padding(.horizontal, 12).padding(.vertical, 10)
                            .contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                    }
                }
                .background(Theme.raised, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
            }
        }
        .padding(.horizontal, 14).padding(.vertical, 12)
        .opacity(card.afford ? 1 : 0.7)
    }

    private var priceText: String {
        let price = card.free ? "Free" : "\(card.cp) CP"
        if !card.afford { return "\(price) · short" }
        return card.reduced ? "\(card.costBase) CP → \(price)" : price
    }

    private func maybeText(_ m: MaybeRoute) -> String {
        var s = m.from
        if let o = m.options { s += " · " + o.joined(separator: " / ") }
        if let who = m.needsOperative { s += " — select the \(who)" } else if !m.condition.isEmpty { s += " — \(m.condition)" }
        return s
    }
}

/// Something in play for the acting operative.
struct ActiveRow: View {
    let card: ActiveCard
    let onEnd: () -> Void
    let onMarkUsed: () -> Void

    var body: some View {
        HStack(alignment: .center, spacing: 10) {
            VStack(alignment: .leading, spacing: 3) {
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Text(card.name).font(.body.weight(.semibold))
                    Text(card.life).font(.caption).foregroundStyle(Theme.muted)
                    if card.disputed { Tag(text: "UNVERIFIED", color: Theme.hot) }
                }
                RuleText(segments: card.hint, color: Color(hex: 0xD1D1D6))
            }
            Spacer(minLength: 4)
            if card.canEnd {
                Button("End", action: onEnd).font(.subheadline).foregroundStyle(Theme.link)
                    .frame(minWidth: 44, minHeight: 44)
            } else if card.canMarkUsed {
                Button("Used", action: onMarkUsed).font(.subheadline).foregroundStyle(Theme.link)
                    .frame(minWidth: 44, minHeight: 44)
                    .accessibilityLabel("Mark \(card.name) used")
            }
        }
        .padding(.leading, 14).padding(.trailing, 8).padding(.vertical, 11)
    }
}
