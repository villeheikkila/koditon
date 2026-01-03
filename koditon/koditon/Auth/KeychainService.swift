import Foundation
import Security

enum KeychainService {
    private static let service = "com.koditon.auth"
    enum Key: String {
        case refreshToken = "refresh_token"
        case accessToken = "access_token"
        case accessTokenExpiresAt = "access_token_expires_at"
        case refreshTokenExpiresAt = "refresh_token_expires_at"
        case userId = "user_id"
        case isAnonymous = "is_anonymous"
    }

    static func save(_ value: String, for key: Key) {
        let data = Data(value.utf8)
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key.rawValue,
        ]
        SecItemDelete(query as CFDictionary)
        var newItem = query
        newItem[kSecValueData as String] = data
        SecItemAdd(newItem as CFDictionary, nil)
    }

    static func load(_ key: Key) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key.rawValue,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        guard status == errSecSuccess, let data = result as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    static func delete(_ key: Key) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key.rawValue,
        ]
        SecItemDelete(query as CFDictionary)
    }

    static func deleteAll() {
        for key in [
            Key.refreshToken, .accessToken, .accessTokenExpiresAt, .refreshTokenExpiresAt, .userId,
            .isAnonymous,
        ] {
            delete(key)
        }
    }
}
