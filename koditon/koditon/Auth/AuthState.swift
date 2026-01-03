import Foundation

enum AuthState: Equatable, Sendable {
    case signedOut
    case anonymous(userId: String)
    case authenticated(userId: String)
    var isSignedIn: Bool {
        switch self {
        case .signedOut: false
        case .anonymous, .authenticated: true
        }
    }

    var userId: String? {
        switch self {
        case .signedOut: nil
        case .anonymous(let userId), .authenticated(let userId): userId
        }
    }
}
