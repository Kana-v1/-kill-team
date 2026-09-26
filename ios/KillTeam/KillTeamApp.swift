import SwiftUI

@main
struct KillTeamApp: App {
    @StateObject private var store = GameStore()

    var body: some Scene {
        WindowGroup {
            GameView()
                .environmentObject(store)
                .preferredColorScheme(.dark)
                .tint(Theme.link)
                .shakeForLog()
        }
    }
}
