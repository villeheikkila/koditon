import AuthenticationServices
import CryptoKit
import OSLog
import SwiftUI

struct SignInWithAppleButtonView: View {
    private let logger = Logger(subsystem: "com.koditon", category: "SignInWithApple")
    @Environment(AppManager.self) private var appManager
    @State private var nonce: String?

    var body: some View {
        SignInWithAppleButton(
            .continue,
            onRequest: { request in
                let nonce = randomString()
                self.nonce = nonce
                request.nonce = sha256(nonce)
                request.requestedScopes = []
            },
            onCompletion: { result in
                Task { await handleAuthorizationResult(result) }
            }
        )
    }

    private func handleAuthorizationResult(_ result: Result<ASAuthorization, Error>) async {
        switch result {
        case .success(let asAuthorization):
            await handleSuccess(asAuthorization: asAuthorization)
        case .failure(let error):
            logger.error("Apple Sign In failed: \(error.localizedDescription)")
        }
    }

    private func handleSuccess(asAuthorization: ASAuthorization) async {
        guard let credential = asAuthorization.credential as? ASAuthorizationAppleIDCredential
        else { return }
        guard let authorizationCodeData = credential.authorizationCode else { return }
        guard let tokenData = credential.identityToken else { return }
        let idToken = String(decoding: tokenData, as: UTF8.self)
        let authorizationCode = String(decoding: authorizationCodeData, as: UTF8.self)
        guard let nonce else { return }
        do {
            try await appManager.auth.signInWithApple(
                .init(
                    authorizationCode: authorizationCode,
                    idToken: idToken,
                    nonce: nonce
                ))
        } catch {
            logger.error("Sign in with Apple failed: \(error.localizedDescription)")
        }
    }

    private func randomString(length: Int = 32) -> String {
        let charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopqrstuvwxyz-._"
        return String((0 ..< length).compactMap { _ in charset.randomElement() })
    }

    private func sha256(_ input: String) -> String {
        let inputData = Data(input.utf8)
        let hashedData = SHA256.hash(data: inputData)
        return hashedData.compactMap { String(format: "%02x", $0) }.joined()
    }
}
