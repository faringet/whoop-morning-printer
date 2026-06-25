import WidgetKit
import SwiftUI

struct MorningStationEntry: TimelineEntry {
let date: Date
let snapshot: NightDisplaySnapshot?
}

struct MorningStationProvider: TimelineProvider {
func placeholder(in context: Context) -> MorningStationEntry {
MorningStationEntry(
date: Date(),
snapshot: .mockArmed
)
}

func getSnapshot(in context: Context, completion: @escaping (MorningStationEntry) -> Void) {
    if context.isPreview {
        completion(
            MorningStationEntry(
                date: Date(),
                snapshot: .mockArmed
            )
        )
        return
    }

    completion(
        MorningStationEntry(
            date: Date(),
            snapshot: SnapshotStore.shared.load()
        )
    )
}

func getTimeline(in context: Context, completion: @escaping (Timeline<MorningStationEntry>) -> Void) {
    Task {
        let now = Date()
        let snapshot = await loadFreshSnapshotOrFallback()

        let entry = MorningStationEntry(
            date: now,
            snapshot: snapshot
        )

        let nextUpdate = Calendar.current.date(
            byAdding: .minute,
            value: 10,
            to: now
        ) ?? now.addingTimeInterval(600)

        completion(
            Timeline(
                entries: [entry],
                policy: .after(nextUpdate)
            )
        )
    }
}

private func loadFreshSnapshotOrFallback() async -> NightDisplaySnapshot? {
    let settings = SharedSettingsStore.shared

    guard settings.hasDisplayToken, let baseURL = settings.baseURL else {
        return SnapshotStore.shared.load()
    }

    do {
        let client = PrinterGatewayClient(
            baseURL: baseURL,
            displayToken: settings.displayToken
        )

        let snapshot = try await client.fetchNightDisplay()
        try? SnapshotStore.shared.save(snapshot)

        return snapshot
    } catch {
        return SnapshotStore.shared.load()
    }
}

}

struct MorningStationWidgetView: View {
let entry: MorningStationEntry

private let stationRed = Color(red: 1.0, green: 0.10, blue: 0.14)

var body: some View {
    ZStack {
        Color.black

        if let snapshot = entry.snapshot {
            if snapshot.isArmed, let wakePlan = snapshot.wakePlan {
                armedView(wakePlan: wakePlan)
            } else {
                notArmedView
            }
        } else {
            noSnapshotView
        }
    }
    .containerBackground(.black, for: .widget)
}

private func armedView(wakePlan: NightDisplayWakePlan) -> some View {
    VStack(alignment: .leading, spacing: 0) {
        HStack(alignment: .center) {
            Text("NEXT\nWAKE")
                .font(.system(size: 11, weight: .semibold, design: .rounded))
                .tracking(1.35)
                .lineSpacing(1)
                .foregroundStyle(stationRed.opacity(0.58))

            Spacer(minLength: 8)

            Text("ARMED")
                .font(.system(size: 9, weight: .semibold, design: .rounded))
                .tracking(1.0)
                .foregroundStyle(stationRed.opacity(0.68))
                .padding(.horizontal, 7)
                .padding(.vertical, 4)
                .background(stationRed.opacity(0.16), in: Capsule())
        }

        Spacer(minLength: 8)

        HStack(alignment: .firstTextBaseline, spacing: 7) {
            Image(systemName: "alarm")
                .font(.system(size: 15, weight: .medium))
                .foregroundStyle(stationRed.opacity(0.52))

            Text(wakeTimeText(wakeAt: wakePlan.wakeAt))
                .font(.system(size: 22, weight: .medium, design: .rounded))
                .monospacedDigit()
                .lineLimit(1)
                .minimumScaleFactor(0.72)
                .foregroundStyle(stationRed.opacity(0.66))
        }

        Spacer(minLength: 8)

        VStack(alignment: .leading, spacing: 2) {
            if countdownHours(wakeAt: wakePlan.wakeAt) > 0 {
                Text(countdownHoursLine(wakeAt: wakePlan.wakeAt))
                    .font(.system(size: 15, weight: .semibold, design: .rounded))
                    .monospacedDigit()
                    .lineLimit(1)
                    .minimumScaleFactor(0.8)
                    .foregroundStyle(stationRed.opacity(0.72))

                Text(countdownMinutesLine(wakeAt: wakePlan.wakeAt))
                    .font(.system(size: 15, weight: .medium, design: .rounded))
                    .monospacedDigit()
                    .lineLimit(1)
                    .minimumScaleFactor(0.8)
                    .foregroundStyle(stationRed.opacity(0.72))
            } else {
                Text(countdownMinutesLine(wakeAt: wakePlan.wakeAt))
                    .font(.system(size: 17, weight: .semibold, design: .rounded))
                    .monospacedDigit()
                    .lineLimit(1)
                    .minimumScaleFactor(0.8)
                    .foregroundStyle(stationRed.opacity(0.92))
            }

            Text("UNTIL MORNING")
                .font(.system(size: 9, weight: .medium, design: .rounded))
                .tracking(1.15)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
                .foregroundStyle(stationRed.opacity(0.44))
                .padding(.top, 2)
        }

        Spacer(minLength: 0)
    }
    .padding(.horizontal, 14)
    .padding(.vertical, 12)
}

private var notArmedView: some View {
    VStack(alignment: .leading, spacing: 0) {
        HStack(alignment: .center) {
            Text("MORNING")
                .font(.system(size: 10, weight: .semibold, design: .rounded))
                .tracking(1.4)
                .foregroundStyle(stationRed.opacity(0.52))

            Spacer(minLength: 8)

            Text("NOT SET")
                .font(.system(size: 9, weight: .semibold, design: .rounded))
                .tracking(1.0)
                .foregroundStyle(stationRed.opacity(0.68))
                .padding(.horizontal, 7)
                .padding(.vertical, 4)
                .background(stationRed.opacity(0.16), in: Capsule())
        }

        Spacer(minLength: 10)

        Text("NO WAKE")
            .font(.system(size: 28, weight: .semibold, design: .rounded))
            .lineLimit(1)
            .minimumScaleFactor(0.7)
            .foregroundStyle(stationRed)

        Text("SCHEDULED")
            .font(.system(size: 17, weight: .semibold, design: .rounded))
            .tracking(1.0)
            .lineLimit(1)
            .minimumScaleFactor(0.75)
            .foregroundStyle(stationRed.opacity(0.78))

        Spacer(minLength: 10)

        Text("OPEN TELEGRAM")
            .font(.system(size: 10, weight: .medium, design: .rounded))
            .tracking(1.1)
            .lineLimit(1)
            .minimumScaleFactor(0.75)
            .foregroundStyle(stationRed.opacity(0.48))

        Spacer(minLength: 0)
    }
    .frame(maxWidth: .infinity, alignment: .leading)
    .padding(.horizontal, 14)
    .padding(.vertical, 12)
}

private var noSnapshotView: some View {
    VStack(alignment: .leading, spacing: 0) {
        HStack(alignment: .center) {
            Text("MORNING")
                .font(.system(size: 10, weight: .semibold, design: .rounded))
                .tracking(1.4)
                .foregroundStyle(stationRed.opacity(0.52))

            Spacer(minLength: 8)

            Text("SYNC")
                .font(.system(size: 9, weight: .semibold, design: .rounded))
                .tracking(1.0)
                .foregroundStyle(stationRed.opacity(0.68))
                .padding(.horizontal, 7)
                .padding(.vertical, 4)
                .background(stationRed.opacity(0.16), in: Capsule())
        }

        Spacer(minLength: 10)

        Text("NO DATA")
            .font(.system(size: 28, weight: .semibold, design: .rounded))
            .lineLimit(1)
            .minimumScaleFactor(0.7)
            .foregroundStyle(stationRed)

        Text("AVAILABLE")
            .font(.system(size: 17, weight: .semibold, design: .rounded))
            .tracking(1.0)
            .lineLimit(1)
            .minimumScaleFactor(0.75)
            .foregroundStyle(stationRed.opacity(0.78))

        Spacer(minLength: 10)

        Text("OPEN APP")
            .font(.system(size: 10, weight: .medium, design: .rounded))
            .tracking(1.1)
            .lineLimit(1)
            .minimumScaleFactor(0.75)
            .foregroundStyle(stationRed.opacity(0.48))

        Spacer(minLength: 0)
    }
    .frame(maxWidth: .infinity, alignment: .leading)
    .padding(.horizontal, 14)
    .padding(.vertical, 12)
}

private func wakeTimeText(wakeAt: Date) -> String {
    let formatter = DateFormatter()
    formatter.dateFormat = "HH:mm"
    return formatter.string(from: wakeAt)
}

private func countdownTotalMinutes(wakeAt: Date) -> Int {
    max(0, Int(wakeAt.timeIntervalSince(entry.date) / 60))
}

private func countdownHours(wakeAt: Date) -> Int {
    countdownTotalMinutes(wakeAt: wakeAt) / 60
}

private func countdownMinutes(wakeAt: Date) -> Int {
    countdownTotalMinutes(wakeAt: wakeAt) % 60
}

private func countdownHoursLine(wakeAt: Date) -> String {
    let hours = countdownHours(wakeAt: wakeAt)

    if hours == 1 {
        return "1 HOUR"
    }

    return "\(hours) HOURS"
}

private func countdownMinutesLine(wakeAt: Date) -> String {
    let minutes = countdownMinutes(wakeAt: wakeAt)

    if countdownHours(wakeAt: wakeAt) == 0 && minutes <= 1 {
        return "SOON"
    }

    if minutes == 1 {
        return "1 MINUTE"
    }

    return "\(minutes) MINUTES"
}

}

struct MorningStationWidget: Widget {
let kind: String = "MorningStationWidget"

var body: some WidgetConfiguration {
    StaticConfiguration(
        kind: kind,
        provider: MorningStationProvider()
    ) { entry in
        MorningStationWidgetView(entry: entry)
    }
    .configurationDisplayName("Morning Station")
    .description("Shows the next morning wake signal.")
    .supportedFamilies([.systemSmall])
    .contentMarginsDisabled()
}

}

private extension NightDisplaySnapshot {
static var mockArmed: NightDisplaySnapshot {
NightDisplaySnapshot(
status: .armed,
wakePlan: NightDisplayWakePlan(
id: 20,
wakeAt: Calendar.current.date(byAdding: .hour, value: 3, to: Date()) ?? Date(),
status: "scheduled",
updatedAt: Date()
),
serverTime: Date(),
timezone: "Europe/Moscow"
)
}

static var mockNotArmed: NightDisplaySnapshot {
    NightDisplaySnapshot(
        status: .notArmed,
        wakePlan: nil,
        serverTime: Date(),
        timezone: "Europe/Moscow"
    )
}

}

#Preview(as: .systemSmall) {
MorningStationWidget()
} timeline: {
MorningStationEntry(
date: Date(),
snapshot: .mockArmed
)

MorningStationEntry(
    date: Date(),
    snapshot: .mockNotArmed
)

MorningStationEntry(
    date: Date(),
    snapshot: nil
)

}
