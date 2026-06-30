import Foundation

enum MorningStationShared {
    static let appGroupIdentifier = "group.com.faringet.morningstation"
    static let nightDisplaySnapshotKey = "nightDisplaySnapshot"
}

enum SnapshotStoreError: LocalizedError {
    case appGroupUnavailable
    case encodingFailed

    var errorDescription: String? {
        switch self {
        case .appGroupUnavailable:
            return "App Group storage is unavailable."
        case .encodingFailed:
            return "Failed to encode night display snapshot."
        }
    }
}

struct SnapshotStore {
    static let shared = SnapshotStore()

    private let userDefaults: UserDefaults?

    init() {
        self.userDefaults = UserDefaults(
            suiteName: MorningStationShared.appGroupIdentifier
        )
    }

    func save(_ snapshot: NightDisplaySnapshot) throws {
        guard let userDefaults else {
            throw SnapshotStoreError.appGroupUnavailable
        }

        let encoder = JSONEncoder()

        do {
            let data = try encoder.encode(snapshot)
            userDefaults.set(data, forKey: MorningStationShared.nightDisplaySnapshotKey)
        } catch {
            throw SnapshotStoreError.encodingFailed
        }
    }

    func load() -> NightDisplaySnapshot? {
        guard let userDefaults else {
            return nil
        }

        guard let data = userDefaults.data(forKey: MorningStationShared.nightDisplaySnapshotKey) else {
            return nil
        }

        return try? JSONDecoder().decode(NightDisplaySnapshot.self, from: data)
    }

    func clear() {
        userDefaults?.removeObject(forKey: MorningStationShared.nightDisplaySnapshotKey)
    }
}
