import Foundation
import KoditonClient
import OSLog

@Observable
@MainActor
final class AuthManager {
    private let logger = Logger(subsystem: "com.koditon", category: "AuthManager")
    private let client: KoditonClient
    private let deviceManager: DeviceManager
    private(set) var state: AuthState = .signedOut
    private var accessToken: String?
    private var accessTokenExpiresAt: Int64?
    private var refreshToken: String?
    private var refreshTokenExpiresAt: Int64?
    private var isRefreshing = false
    private var refreshContinuations: [CheckedContinuation<String, Error>] = []

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
        } else {
            do {
                _ = try await getValidAccessToken()
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
            }
        case .default(let statusCode, _):
            throw AuthError.signInFailed("Sign in failed with status: \(statusCode)")
        }
    }

    func signOut() async {
        logger.debug("Signing out")
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

    private func refreshTokens() async throws -> String {
        if isRefreshing {
            return try await withCheckedThrowingContinuation { continuation in
                refreshContinuations.append(continuation)
            }
        }
        isRefreshing = true
        defer {
            isRefreshing = false
            refreshContinuations.removeAll()
        }
        guard let refreshToken = refreshToken ?? KeychainService.load(.refreshToken) else {
            throw AuthError.notAuthenticated
        }
        logger.debug("Refreshing tokens")
        do {
            let response = try await client.api.authRefreshTokens(
                .init(body: .json(.init(refreshToken: refreshToken)))
            )
            switch response {
            case .ok(let ok):
                switch ok.body {
                case .json(let body):
                    let token = body.accessToken
                    accessToken = token
                    accessTokenExpiresAt = body.accessTokenExpiresAt
                    self.refreshToken = body.refreshToken
                    refreshTokenExpiresAt = body.refreshTokenExpiresAt
                    persistTokens(userId: body.userId)
                    for continuation in refreshContinuations {
                        continuation.resume(returning: token)
                    }
                    return token
                }
            case .default(let statusCode, _):
                let error = AuthError.refreshFailed("Refresh failed with status: \(statusCode)")
                for continuation in refreshContinuations {
                    continuation.resume(throwing: error)
                }
                if statusCode == 401 {
                    clearTokens()
                    state = .signedOut
                }
                throw error
            }
        } catch {
            for continuation in refreshContinuations {
                continuation.resume(throwing: error)
            }
            throw error
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
