import WidgetKit
import SwiftUI

struct MorningStationEntry: TimelineEntry {
let date: Date
let wakeAt: Date
let isArmed: Bool
}

struct MorningStationProvider: TimelineProvider {
func placeholder(in context: Context) -> MorningStationEntry {
MorningStationEntry(
date: Date(),
wakeAt: mockWakeAt(),
isArmed: true
)
}

func getSnapshot(in context: Context, completion: @escaping (MorningStationEntry) -> Void) {
    completion(
        MorningStationEntry(
            date: Date(),
            wakeAt: mockWakeAt(),
            isArmed: true
        )
    )
}

func getTimeline(in context: Context, completion: @escaping (Timeline<MorningStationEntry>) -> Void) {
    let now = Date()
    let entry = MorningStationEntry(
        date: now,
        wakeAt: mockWakeAt(),
        isArmed: true
    )

    let nextUpdate = Calendar.current.date(byAdding: .minute, value: 5, to: now) ?? now.addingTimeInterval(300)
    completion(Timeline(entries: [entry], policy: .after(nextUpdate)))
}

private func mockWakeAt() -> Date {
    var components = Calendar.current.dateComponents([.year, .month, .day], from: Date())
    components.hour = 8
    components.minute = 0
    components.second = 0

    let todayWake = Calendar.current.date(from: components) ?? Date().addingTimeInterval(3600)

    if todayWake > Date() {
        return todayWake
    }

    return Calendar.current.date(byAdding: .day, value: 1, to: todayWake) ?? todayWake.addingTimeInterval(86400)
}

}

struct MorningStationWidgetView: View {
let entry: MorningStationEntry

private let stationRed = Color(red: 1.0, green: 0.10, blue: 0.14)

var body: some View {
    ZStack {
        Color.black

        if entry.isArmed {
            armedView
        } else {
            notArmedView
        }
    }
    .containerBackground(.black, for: .widget)
}

private var armedView: some View {
    VStack(alignment: .leading, spacing: 0) {
        HStack(alignment: .center) {
            Text("WAKE")
                .font(.system(size: 11, weight: .semibold, design: .rounded))
                .tracking(1.6)
                .foregroundStyle(stationRed.opacity(0.58))

            Spacer(minLength: 8)

            Text("ARMED")
                .font(.system(size: 9, weight: .semibold, design: .rounded))
                .tracking(1.1)
                .foregroundStyle(stationRed.opacity(0.68))
                .padding(.horizontal, 7)
                .padding(.vertical, 4)
                .background(stationRed.opacity(0.16), in: Capsule())
        }

        Spacer(minLength: 8)

        Text(wakeTimeText)
            .font(.system(size: 39, weight: .semibold, design: .rounded))
            .monospacedDigit()
            .lineLimit(1)
            .minimumScaleFactor(0.72)
            .foregroundStyle(stationRed)

        Spacer(minLength: 8)

        VStack(alignment: .leading, spacing: 2) {
            Text(timeUntilWakeText)
                .font(.system(size: 23, weight: .semibold, design: .rounded))
                .monospacedDigit()
                .lineLimit(1)
                .minimumScaleFactor(0.75)
                .foregroundStyle(stationRed.opacity(0.92))

            Text("UNTIL MORNING")
                .font(.system(size: 9, weight: .medium, design: .rounded))
                .tracking(1.05)
                .lineLimit(1)
                .minimumScaleFactor(0.75)
                .foregroundStyle(stationRed.opacity(0.44))
        }

        Spacer(minLength: 0)
    }
    .padding(.horizontal, 14)
    .padding(.vertical, 12)
}

private var notArmedView: some View {
    VStack(alignment: .leading, spacing: 0) {
        Text("MORNING")
            .font(.system(size: 11, weight: .semibold, design: .rounded))
            .tracking(1.5)
            .foregroundStyle(stationRed.opacity(0.52))

        Spacer(minLength: 8)

        Text("NOT\nARMED")
            .font(.system(size: 25, weight: .semibold, design: .rounded))
            .lineSpacing(1)
            .multilineTextAlignment(.leading)
            .foregroundStyle(stationRed.opacity(0.9))

        Spacer(minLength: 8)

        Text("SET WAKE\nIN TELEGRAM")
            .font(.system(size: 9, weight: .medium, design: .rounded))
            .tracking(1.0)
            .lineSpacing(2)
            .multilineTextAlignment(.leading)
            .foregroundStyle(stationRed.opacity(0.42))
    }
    .frame(maxWidth: .infinity, alignment: .leading)
    .padding(.horizontal, 14)
    .padding(.vertical, 12)
}

private var wakeTimeText: String {
    let formatter = DateFormatter()
    formatter.dateFormat = "HH:mm"
    return formatter.string(from: entry.wakeAt)
}

    private var timeUntilWakeText: String {
        let totalMinutes = max(0, Int(entry.wakeAt.timeIntervalSince(entry.date) / 60))

        if totalMinutes >= 90 {
            let roundedHours = max(1, (totalMinutes + 30) / 60)

            if roundedHours == 1 {
                return "1 HOUR"
            }

            return "\(roundedHours) HOURS"
        }

        if totalMinutes >= 60 {
            return "1 HOUR"
        }

        return "\(totalMinutes) MIN"
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

#Preview(as: .systemSmall) {
MorningStationWidget()
} timeline: {
MorningStationEntry(
date: Date(),
wakeAt: Calendar.current.date(byAdding: .hour, value: 8, to: Date()) ?? Date(),
isArmed: true
)

MorningStationEntry(
    date: Date(),
    wakeAt: Calendar.current.date(byAdding: .hour, value: 8, to: Date()) ?? Date(),
    isArmed: false
)

}

