import Foundation

struct SharedSettingsStore {
    static let shared = SharedSettingsStore()

    private enum Keys {
        static let baseURL = "printerGatewayBaseURL"
        static let displayToken = "printerGatewayDisplayToken"
    }

    private let userDefaults: UserDefaults?

    init() {
        self.userDefaults = UserDefaults(
            suiteName: MorningStationShared.appGroupIdentifier
        )
    }

    var baseURLText: String {
        userDefaults?.string(forKey: Keys.baseURL) ?? "https://printergateway.veloranet.ru"
    }

    var displayToken: String {
        userDefaults?.string(forKey: Keys.displayToken) ?? ""
    }

    var baseURL: URL? {
        URL(string: baseURLText.trimmingCharacters(in: .whitespacesAndNewlines))
    }

    var hasDisplayToken: Bool {
        !displayToken.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    func save(baseURLText: String, displayToken: String) {
        userDefaults?.set(
            baseURLText.trimmingCharacters(in: .whitespacesAndNewlines),
            forKey: Keys.baseURL
        )

        userDefaults?.set(
            displayToken.trimmingCharacters(in: .whitespacesAndNewlines),
            forKey: Keys.displayToken
        )
    }

    func clear() {
        userDefaults?.removeObject(forKey: Keys.baseURL)
        userDefaults?.removeObject(forKey: Keys.displayToken)
    }
}
