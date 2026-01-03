import Foundation
import KoditonClient
import OSLog

@Observable
@MainActor
final class AppManager {
    private static let baseURL = URL(string: "http://localhost:8080")!
    private let logger = Logger(subsystem: "com.koditon", category: "AppManager")
    let deviceManager: DeviceManager
    let auth: AuthManager
    let client: KoditonClient

    private init(deviceManager: DeviceManager, auth: AuthManager, client: KoditonClient) {
        self.deviceManager = deviceManager
        self.auth = auth
        self.client = client
    }

    static func create() async -> AppManager {
        let deviceManager = DeviceManager()
        let authClient = KoditonClient(baseURL: baseURL)
        let auth = AuthManager(client: authClient, deviceManager: deviceManager)
        let authMiddleware = AuthMiddleware(
            tokenProvider: { [auth] in
                try await auth.getValidAccessToken()
            },
            deviceID: deviceManager.deviceID
        )
        let client = KoditonClient(baseURL: baseURL, middlewares: [authMiddleware])
        let appManager = AppManager(deviceManager: deviceManager, auth: auth, client: client)
        await auth.restoreSession()
        return appManager
    }
}
