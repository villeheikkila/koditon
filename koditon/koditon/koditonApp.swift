import SwiftUI

@main
struct koditonApp: App {
    @State private var appManager: AppManager?

    var body: some Scene {
        WindowGroup {
            Group {
                if let appManager {
                    ContentView()
                        .environment(appManager)
                } else {
                    ProgressView("Loading...")
                }
            }
            .task {
                if appManager == nil {
                    appManager = await AppManager.create()
                }
            }
        }
    }
}
