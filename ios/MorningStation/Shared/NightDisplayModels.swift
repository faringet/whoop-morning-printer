import Foundation

enum NightDisplayStatus: String, Codable, Equatable {
    case armed
    case notArmed = "not_armed"
    case unknown

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        let rawValue = try container.decode(String.self)
        self = NightDisplayStatus(rawValue: rawValue) ?? .unknown
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        try container.encode(rawValue)
    }
}

struct NightDisplaySnapshot: Codable, Equatable {
    let status: NightDisplayStatus
    let wakePlan: NightDisplayWakePlan?
    let serverTime: Date
    let timezone: String

    var isArmed: Bool {
        status == .armed && wakePlan != nil
    }

    enum CodingKeys: String, CodingKey {
        case status
        case wakePlan = "wake_plan"
        case serverTime = "server_time"
        case timezone
    }

    init(
        status: NightDisplayStatus,
        wakePlan: NightDisplayWakePlan?,
        serverTime: Date,
        timezone: String
    ) {
        self.status = status
        self.wakePlan = wakePlan
        self.serverTime = serverTime
        self.timezone = timezone
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)

        status = try container.decode(NightDisplayStatus.self, forKey: .status)
        wakePlan = try container.decodeIfPresent(NightDisplayWakePlan.self, forKey: .wakePlan)

        let serverTimeRaw = try container.decode(String.self, forKey: .serverTime)
        serverTime = try Date.parseNightDisplayDate(serverTimeRaw)

        timezone = try container.decode(String.self, forKey: .timezone)
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)

        try container.encode(status, forKey: .status)
        try container.encodeIfPresent(wakePlan, forKey: .wakePlan)
        try container.encode(serverTime.nightDisplayISOString, forKey: .serverTime)
        try container.encode(timezone, forKey: .timezone)
    }
}

struct NightDisplayWakePlan: Codable, Equatable {
    let id: Int64
    let wakeAt: Date
    let status: String
    let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case wakeAt = "wake_at"
        case status
        case updatedAt = "updated_at"
    }

    init(
        id: Int64,
        wakeAt: Date,
        status: String,
        updatedAt: Date
    ) {
        self.id = id
        self.wakeAt = wakeAt
        self.status = status
        self.updatedAt = updatedAt
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)

        id = try container.decode(Int64.self, forKey: .id)

        let wakeAtRaw = try container.decode(String.self, forKey: .wakeAt)
        wakeAt = try Date.parseNightDisplayDate(wakeAtRaw)

        status = try container.decode(String.self, forKey: .status)

        let updatedAtRaw = try container.decode(String.self, forKey: .updatedAt)
        updatedAt = try Date.parseNightDisplayDate(updatedAtRaw)
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)

        try container.encode(id, forKey: .id)
        try container.encode(wakeAt.nightDisplayISOString, forKey: .wakeAt)
        try container.encode(status, forKey: .status)
        try container.encode(updatedAt.nightDisplayISOString, forKey: .updatedAt)
    }
}

extension Date {
    static func parseNightDisplayDate(_ value: String) throws -> Date {
        let normalizedValue = normalizeISO8601FractionalSeconds(value)

        let formatterWithFractionalSeconds = ISO8601DateFormatter()
        formatterWithFractionalSeconds.formatOptions = [
            .withInternetDateTime,
            .withFractionalSeconds
        ]

        if let date = formatterWithFractionalSeconds.date(from: normalizedValue) {
            return date
        }

        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [
            .withInternetDateTime
        ]

        if let date = formatter.date(from: normalizedValue) {
            return date
        }

        throw DecodingError.dataCorrupted(
            DecodingError.Context(
                codingPath: [],
                debugDescription: "Invalid night display date: \(value)"
            )
        )
    }

    var nightDisplayISOString: String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [
            .withInternetDateTime,
            .withFractionalSeconds
        ]

        return formatter.string(from: self)
    }

    private static func normalizeISO8601FractionalSeconds(_ value: String) -> String {
        guard let dotIndex = value.firstIndex(of: ".") else {
            return value
        }

        let afterDot = value.index(after: dotIndex)
        let suffixIndex = value[afterDot...].firstIndex { character in
            character == "Z" || character == "+" || character == "-"
        }

        guard let suffixIndex else {
            return value
        }

        let fraction = value[afterDot..<suffixIndex]
        let normalizedFraction = String(fraction.prefix(3)).padding(
            toLength: 3,
            withPad: "0",
            startingAt: 0
        )

        return String(value[..<dotIndex]) + "." + normalizedFraction + String(value[suffixIndex...])
    }
}
