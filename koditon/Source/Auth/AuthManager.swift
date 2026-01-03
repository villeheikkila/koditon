import Foundation
import KoditonClient
import OSLog

@Observable
@MainActor
final class AuthManager {
    private static let autoRefreshInterval: Duration = .seconds(10)
    private static let refreshThreshold: Duration = .seconds(30)
    private let logger = Logger(subsystem: "com.koditon", category: "AuthManager")
    private let client: KoditonClient
    private let deviceManager: DeviceManager
    private(set) var state: AuthState = .signedOut
    private var accessToken: String?
    private var accessTokenExpiresAt: Int64?
    private var refreshToken: String?
    private var refreshTokenExpiresAt: Int64?
    private var inFlightRefresh: Task<String, Error>?
    private var autoRefreshTask: Task<Void, Never>?

    init(client: KoditonClient, deviceManager: DeviceManager) {
        self.client = client
        self.deviceManager = deviceManager
    }

    func restoreSession() async {
        guard let refreshToken = KeychainService.load(.refreshToken),
              let expiresAtString = KeychainService.load(.refreshTokenExpiresAt),
              let expiresAt = Int64(expiresAtString),
              expiresAt > Int64(Date().timeIntervalSince1970)
        else {
            logger.debug("No valid session to restore")
            clearTokens()
            return
        }
        self.refreshToken = refreshToken
        refreshTokenExpiresAt = expiresAt
        if let accessToken = KeychainService.load(.accessToken),
           let accessExpiresAtString = KeychainService.load(.accessTokenExpiresAt),
           let accessExpiresAt = Int64(accessExpiresAtString),
           accessExpiresAt > Int64(Date().timeIntervalSince1970) + 60
        {
            self.accessToken = accessToken
            accessTokenExpiresAt = accessExpiresAt
        }
        if let userId = KeychainService.load(.userId) {
            let isAnonymous = KeychainService.load(.isAnonymous) == "true"
            state = isAnonymous ? .anonymous(userId: userId) : .authenticated(userId: userId)
            logger.info("Session restored for user: \(userId, privacy: .private)")
            startAutoRefresh()
        } else {
            do {
                _ = try await getValidAccessToken()
                startAutoRefresh()
            } catch {
                logger.error("Failed to refresh during restore: \(error.localizedDescription)")
                clearTokens()
            }
        }
    }

    func signInAnonymously() async throws {
        logger.debug("Signing in anonymously")
        let response = try await client.api.authSignInAnonymous(
            .init(
                headers: .init(xDeviceID: deviceManager.deviceID),
                body: .json(.init())
            )
        )
        switch response {
        case .ok(let ok):
            switch ok.body {
            case .json(let body):
                handleAuthResponse(body, isAnonymous: true)
                startAutoRefresh()
            }
        case .default(let statusCode, _):
            throw AuthError.signInFailed("Sign in failed with status: \(statusCode)")
        }
    }

    struct AppleSignInCredential {
        let authorizationCode: String
        let idToken: String
        let nonce: String
    }

    func signInWithApple(_ credential: AppleSignInCredential) async throws {
        logger.debug("Signing in with Apple")
        let response = try await client.api.authSignInWithApple(
            .init(
                headers: .init(xDeviceID: deviceManager.deviceID),
                body: .json(
                    .init(authorizationCode: credential.authorizationCode, nonce: credential.nonce))
            )
        )
        switch response {
        case .ok(let ok):
            switch ok.body {
            case .json(let body):
                handleAuthResponse(body, isAnonymous: false)
                startAutoRefresh()
            }
        case .default(let statusCode, _):
            throw AuthError.signInFailed("Sign in failed with status: \(statusCode)")
        }
    }

    func signOut() async {
        logger.debug("Signing out")
        stopAutoRefresh()
        if accessToken != nil {
            do {
                let response = try await client.api.authSignOut(.init())
                switch response {
                case .ok:
                    logger.debug("Sign out successful")
                case .default:
                    logger.warning("Sign out request failed, clearing local state anyway")
                }
            } catch {
                logger.warning("Sign out request failed: \(error.localizedDescription)")
            }
        }
        clearTokens()
        state = .signedOut
    }

    func getValidAccessToken() async throws -> String {
        if let token = accessToken,
           let expiresAt = accessTokenExpiresAt,
           expiresAt > Int64(Date().timeIntervalSince1970) + 60
        {
            return token
        }
        return try await refreshTokens()
    }

    func refreshTokensForRetry() async throws -> String {
        try await refreshTokens()
    }

    private func refreshTokens() async throws -> String {
        if let inFlight = inFlightRefresh {
            return try await inFlight.value
        }
        let task = Task { @MainActor in
            defer { inFlightRefresh = nil }
            guard let currentRefreshToken = refreshToken ?? KeychainService.load(.refreshToken)
            else {
                throw AuthError.notAuthenticated
            }
            logger.debug("Refreshing tokens")
            let response = try await client.api.authRefreshTokens(
                .init(body: .json(.init(refreshToken: currentRefreshToken)))
            )
            switch response {
            case .ok(let ok):
                switch ok.body {
                case .json(let body):
                    accessToken = body.accessToken
                    accessTokenExpiresAt = body.accessTokenExpiresAt
                    refreshToken = body.refreshToken
                    refreshTokenExpiresAt = body.refreshTokenExpiresAt
                    persistTokens(userId: body.userId)
                    return body.accessToken
                }
            case .default(let statusCode, _):
                if statusCode == 401 {
                    stopAutoRefresh()
                    clearTokens()
                    state = .signedOut
                }
                throw AuthError.refreshFailed("Refresh failed with status: \(statusCode)")
            }
        }
        inFlightRefresh = task
        return try await task.value
    }

    private func startAutoRefresh() {
        guard autoRefreshTask == nil else { return }
        logger.debug("Starting auto-refresh")
        autoRefreshTask = Task {
            while !Task.isCancelled {
                try? await Task.sleep(for: Self.autoRefreshInterval)
                guard !Task.isCancelled else { break }
                await autoRefreshTickIfNeeded()
            }
        }
    }

    private func stopAutoRefresh() {
        logger.debug("Stopping auto-refresh")
        autoRefreshTask?.cancel()
        autoRefreshTask = nil
    }

    private func autoRefreshTickIfNeeded() async {
        guard let expiresAt = accessTokenExpiresAt else { return }
        let now = Int64(Date().timeIntervalSince1970)
        let secondsUntilExpiry = expiresAt - now
        let threshold = Int64(Self.refreshThreshold.components.seconds)
        guard secondsUntilExpiry <= threshold else {
            logger.debug("Token expires in \(secondsUntilExpiry)s, no refresh needed")
            return
        }
        logger.debug("Token expires in \(secondsUntilExpiry)s, refreshing proactively")
        do {
            _ = try await refreshTokens()
            logger.debug("Auto-refresh succeeded")
        } catch {
            logger.warning("Auto-refresh failed: \(error.localizedDescription)")
        }
    }

    private func handleAuthResponse(
        _ response: Components.Schemas.AuthTokensResponse, isAnonymous: Bool
    ) {
        accessToken = response.accessToken
        accessTokenExpiresAt = response.accessTokenExpiresAt
        refreshToken = response.refreshToken
        refreshTokenExpiresAt = response.refreshTokenExpiresAt
        let userId = response.userId
        persistTokens(userId: userId, isAnonymous: isAnonymous)
        state = isAnonymous ? .anonymous(userId: userId) : .authenticated(userId: userId)
        logger.info(
            "Signed in as \(isAnonymous ? "anonymous" : "authenticated") user: \(userId, privacy: .private)"
        )
    }

    private func persistTokens(userId: String, isAnonymous: Bool? = nil) {
        if let accessToken {
            KeychainService.save(accessToken, for: .accessToken)
        }
        if let accessTokenExpiresAt {
            KeychainService.save(String(accessTokenExpiresAt), for: .accessTokenExpiresAt)
        }
        if let refreshToken {
            KeychainService.save(refreshToken, for: .refreshToken)
        }
        if let refreshTokenExpiresAt {
            KeychainService.save(String(refreshTokenExpiresAt), for: .refreshTokenExpiresAt)
        }
        KeychainService.save(userId, for: .userId)
        if let isAnonymous {
            KeychainService.save(isAnonymous ? "true" : "false", for: .isAnonymous)
        }
    }

    private func clearTokens() {
        accessToken = nil
        accessTokenExpiresAt = nil
        refreshToken = nil
        refreshTokenExpiresAt = nil
        KeychainService.deleteAll()
    }
}

enum AuthError: LocalizedError {
    case notAuthenticated
    case signInFailed(String)
    case refreshFailed(String)
    var errorDescription: String? {
        switch self {
        case .notAuthenticated: "Not authenticated"
        case .signInFailed(let message): "Sign in failed: \(message)"
        case .refreshFailed(let message): "Token refresh failed: \(message)"
        }
    }
}
