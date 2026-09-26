import SwiftUI
import UIKit

/// A rolling diagnostic log kept on the phone. Shake the phone to read, copy or
/// share it: on a device with no debugger attached, this is the only way to see
/// why something (loading rules, saving a game) went wrong.
enum Log {
    private static let queue = DispatchQueue(label: "killteam.log")

    static let url: URL = {
        let dir = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir.appendingPathComponent("killteam-log.txt")
    }()

    /// Thread-safe; call from anywhere.
    static func write(_ message: String, _ area: String = "app") {
        let stamp = Date()
        queue.async {
            let f = DateFormatter()
            f.dateFormat = "MM-dd HH:mm:ss.SSS"
            let data = Data("\(f.string(from: stamp)) [\(area)] \(message)\n".utf8)
            if let h = try? FileHandle(forWritingTo: url) {
                _ = try? h.seekToEnd()
                try? h.write(contentsOf: data)
                try? h.close()
            } else {
                try? data.write(to: url)
            }
            // keep the file small: past ~600 KB, keep the newest ~300 KB
            if let size = (try? FileManager.default.attributesOfItem(atPath: url.path))?[.size] as? Int,
               size > 600_000, let all = try? Data(contentsOf: url) {
                try? Data(all.suffix(300_000)).write(to: url)
            }
        }
    }

    static func read() -> String {
        queue.sync {
            guard let d = try? Data(contentsOf: url) else { return "" }
            return String(decoding: d, as: UTF8.self)
        }
    }

    static func clear() {
        queue.sync { try? FileManager.default.removeItem(at: url) }
    }
}

extension Notification.Name {
    static let deviceDidShake = Notification.Name("killteam.deviceDidShake")
}

extension UIWindow {
    open override func motionEnded(_ motion: UIEvent.EventSubtype, with event: UIEvent?) {
        if motion == .motionShake {
            NotificationCenter.default.post(name: .deviceDidShake, object: nil)
        }
        super.motionEnded(motion, with: event)
    }
}

/// Shake to open the log. Put on every screen that can be on top: a sheet can
/// only be presented from the view that is currently frontmost.
struct ShakeForLog: ViewModifier {
    @State private var showing = false

    func body(content: Content) -> some View {
        content
            .onReceive(NotificationCenter.default.publisher(for: .deviceDidShake)) { _ in showing = true }
            .sheet(isPresented: $showing) { LogView() }
    }
}

extension View {
    func shakeForLog() -> some View { modifier(ShakeForLog()) }
}

struct LogView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var text = ""
    @State private var copied = false

    var body: some View {
        NavigationStack {
            ScrollViewReader { proxy in
                ScrollView {
                    Text(text.isEmpty ? "Nothing logged yet." : text)
                        .font(.system(size: 11, design: .monospaced))
                        .textSelection(.enabled)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(12)
                    Color.clear.frame(height: 1).id("end")
                }
                .onChange(of: text) { _, _ in proxy.scrollTo("end", anchor: .bottom) }
            }
            .onAppear { text = Log.read() }
            .navigationTitle("Log")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Close") { dismiss() }
                }
                ToolbarItemGroup(placement: .primaryAction) {
                    Button(copied ? "Copied" : "Copy") {
                        UIPasteboard.general.string = text
                        copied = true
                    }
                    ShareLink(item: Log.url) { Image(systemName: "square.and.arrow.up") }
                }
                ToolbarItem(placement: .bottomBar) {
                    Button("Clear", role: .destructive) {
                        Log.clear()
                        text = ""
                    }
                }
            }
        }
        .preferredColorScheme(.dark)
    }
}
