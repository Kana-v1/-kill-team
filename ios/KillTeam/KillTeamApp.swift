import SwiftUI

@main
struct KillTeamApp: App {
    @StateObject private var store = GameStore()
    @Environment(\.scenePhase) private var scenePhase

    var body: some Scene {
        WindowGroup {
            GameView()
                .environmentObject(store)
                .preferredColorScheme(.dark)
                .tint(Theme.link)
                .shakeForLog()
        }
        .onChange(of: scenePhase) { _, phase in
            // Photos dropped into the app's folder (Files app) are picked up on return.
            if phase == .active { store.importDroppedPhotos() }
        }
    }
}
