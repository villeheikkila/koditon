import Foundation

@Observable
final class DeviceManager: Sendable {
    private static let deviceIDKey = "koditon.device.id"
    let deviceID: String

    init() {
        if let existing = UserDefaults.standard.string(forKey: Self.deviceIDKey) {
            deviceID = existing
        } else {
            let newID = UUID().uuidString
            UserDefaults.standard.set(newID, forKey: Self.deviceIDKey)
            deviceID = newID
        }
    }
}
