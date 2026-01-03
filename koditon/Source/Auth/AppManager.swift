import Foundation
import KoditonClient
import OSLog
import Observation

@Observable
@MainActor
final class AppManager {
    private let logger = Logger(subsystem: "com.koditon", category: "AppManager")
    let deviceManager: DeviceManager
    let auth: AuthManager
    let client: KoditonClient
    private(set) var featureFlags: Set<String> = []
    private var stateObservation: Task<Void, Never>?

    private init(deviceManager: DeviceManager, auth: AuthManager, client: KoditonClient) {
        self.deviceManager = deviceManager
        self.auth = auth
        self.client = client
    }

    static func create() async -> AppManager {
        let deviceManager = DeviceManager()
        let authClient = KoditonClient(baseURL: BuildConfig.apiBaseURL)
        let auth = AuthManager(client: authClient, deviceManager: deviceManager)
        let authMiddleware = AuthMiddleware(
            tokenProvider: { [auth] in
                try await auth.getValidAccessToken()
            },
            deviceID: deviceManager.deviceID,
            onUnauthorized: { [auth] in
                try await auth.refreshTokensForRetry()
            }
        )
        let client = KoditonClient(baseURL: BuildConfig.apiBaseURL, middlewares: [authMiddleware])
        let appManager = AppManager(deviceManager: deviceManager, auth: auth, client: client)
        appManager.startObservingAuthState()
        await auth.restoreSession()
        return appManager
    }

    func hasFeatureFlag(_ flag: String) -> Bool {
        featureFlags.contains(flag)
    }

    private func startObservingAuthState() {
        stateObservation = Task { [weak self] in
            guard let self else { return }
            let states = Observations { self.auth.state }
            var previousState: AuthState?
            for await state in states {
                guard !Task.isCancelled else { break }
                if state != previousState {
                    await handleAuthStateChange(to: state)
                    previousState = state
                }
            }
        }
    }

    private func handleAuthStateChange(to state: AuthState) async {
        switch state {
        case .signedOut:
            featureFlags = []
            logger.debug("Cleared feature flags on sign out")
        case .anonymous, .authenticated:
            await loadCurrentUser()
        }
    }

    private func loadCurrentUser() async {
        do {
            let response = try await client.api.authGetCurrentUser()
            switch response {
            case .ok(let ok):
                switch ok.body {
                case .json(let body):
                    featureFlags = Set(body.featureFlags ?? [])
                    logger.debug("Loaded feature flags: \(self.featureFlags)")
                }
            case .default(let statusCode, _):
                logger.warning("Failed to load current user: status \(statusCode)")
            }
        } catch {
            logger.warning("Failed to load current user: \(error.localizedDescription)")
        }
    }
}
