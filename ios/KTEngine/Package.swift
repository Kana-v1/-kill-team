// swift-tools-version:5.9
// The rules engine: pure Foundation, so it builds and tests on Linux (`swift test`)
// as well as in the iOS app. Swift 5 language mode on purpose — the app's only
// compiler is CI (see ro_podcasts' reasoning), so no strict-concurrency surprises.
import PackageDescription

let package = Package(
    name: "KTEngine",
    platforms: [.iOS(.v17), .macOS(.v13)],
    products: [.library(name: "KTEngine", targets: ["KTEngine"])],
    targets: [
        .target(name: "KTEngine"),
        .testTarget(name: "KTEngineTests", dependencies: ["KTEngine"]),
    ],
    swiftLanguageVersions: [.v5]
)
