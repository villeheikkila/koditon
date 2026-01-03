import Foundation
import KoditonClient
import OSLog
import Observation

struct AvailableMunicipality: Identifiable, Hashable, Sendable {
    let id: String
    let code: String
    let nameFi: String
    let nameSv: String?
}

struct AvailablePostalCode: Identifiable, Hashable, Sendable {
    let id: String
    let code: String
    let nameFi: String
    let nameSv: String?
    let municipalityId: String
}

struct TranslatedValue: Identifiable, Hashable, Sendable {
    let value: String
    let translation: String
    var id: String { value }
    var displayName: String { translation }
}

struct AvailabilityData: Sendable {
    let municipalities: [AvailableMunicipality]
    let postalCodes: [AvailablePostalCode]
    let categories: [TranslatedValue]
    let types: [TranslatedValue]
    static let empty = AvailabilityData(
        municipalities: [], postalCodes: [], categories: [], types: [])
    func postalCodes(for municipalityId: String) -> [AvailablePostalCode] {
        postalCodes.filter { $0.municipalityId == municipalityId }
    }
    func categoryDisplayName(for value: String) -> String {
        categories.first { $0.value == value }?.translation ?? value
    }
    func typeDisplayName(for value: String) -> String {
        types.first { $0.value == value }?.translation ?? value
    }
}

@Observable
@MainActor
final class AppManager {
    private let logger = Logger(subsystem: "com.koditon", category: "AppManager")
    let deviceManager: DeviceManager
    let auth: AuthManager
    let client: KoditonClient
    private(set) var featureFlags: Set<String> = []
    private(set) var availability: AvailabilityData = .empty
    private(set) var isLoadingAvailability = false
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
            availability = .empty
            logger.debug("Cleared feature flags and availability on sign out")
        case .anonymous, .authenticated:
            await loadCurrentUser()
            await loadAvailability()
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
    private func loadAvailability() async {
        isLoadingAvailability = true
        defer { isLoadingAvailability = false }
        async let locationsTask = loadLocations()
        async let categoriesTask = loadCategories()
        async let typesTask = loadTypes()
        let (locations, categories, types) = await (locationsTask, categoriesTask, typesTask)
        availability = AvailabilityData(
            municipalities: locations.municipalities,
            postalCodes: locations.postalCodes,
            categories: categories,
            types: types
        )
        logger.debug(
            "Loaded availability: \(self.availability.municipalities.count) municipalities, \(self.availability.postalCodes.count) postal codes, \(self.availability.categories.count) categories, \(self.availability.types.count) types"
        )
    }
    private func loadLocations() async -> (
        municipalities: [AvailableMunicipality], postalCodes: [AvailablePostalCode]
    ) {
        do {
            let response = try await client.api.availabilityLocations()
            switch response {
            case .ok(let ok):
                switch ok.body {
                case .json(let body):
                    let municipalities = (body.municipalities ?? []).map { m in
                        AvailableMunicipality(
                            id: m.id,
                            code: m.code,
                            nameFi: m.nameFi,
                            nameSv: m.nameSv
                        )
                    }
                    let postalCodes = (body.postalCodes ?? []).map { p in
                        AvailablePostalCode(
                            id: p.id,
                            code: p.code,
                            nameFi: p.nameFi,
                            nameSv: p.nameSv,
                            municipalityId: p.municipalityId
                        )
                    }
                    return (municipalities, postalCodes)
                }
            case .default(let statusCode, _):
                logger.warning("Failed to load locations: status \(statusCode)")
            }
        } catch {
            logger.warning("Failed to load locations: \(error.localizedDescription)")
        }
        return ([], [])
    }
    private func loadCategories() async -> [TranslatedValue] {
        do {
            let response = try await client.api.availabilityCategories()
            switch response {
            case .ok(let ok):
                switch ok.body {
                case .json(let body):
                    return (body.categories ?? []).map { c in
                        TranslatedValue(value: c.value, translation: c.translation)
                    }
                }
            case .default(let statusCode, _):
                logger.warning("Failed to load categories: status \(statusCode)")
            }
        } catch {
            logger.warning("Failed to load categories: \(error.localizedDescription)")
        }
        return []
    }
    private func loadTypes() async -> [TranslatedValue] {
        do {
            let response = try await client.api.availabilityTypes()
            switch response {
            case .ok(let ok):
                switch ok.body {
                case .json(let body):
                    return (body.types ?? []).map { t in
                        TranslatedValue(value: t.value, translation: t.translation)
                    }
                }
            case .default(let statusCode, _):
                logger.warning("Failed to load types: status \(statusCode)")
            }
        } catch {
            logger.warning("Failed to load types: \(error.localizedDescription)")
        }
        return []
    }
    func refreshAvailability() async {
        await loadAvailability()
    }
}
