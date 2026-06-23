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
                HStack(spacing: 5) {
                    Image(systemName: "alarm.fill")
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(stationRed.opacity(0.55))

                    Text("NEXT WAKE")
                        .font(.system(size: 10, weight: .semibold, design: .rounded))
                        .tracking(1.25)
                        .foregroundStyle(stationRed.opacity(0.55))
                }

                Spacer(minLength: 8)

                Text("ARMED")
                    .font(.system(size: 9, weight: .semibold, design: .rounded))
                    .tracking(1.0)
                    .foregroundStyle(stationRed.opacity(0.68))
                    .padding(.horizontal, 7)
                    .padding(.vertical, 4)
                    .background(stationRed.opacity(0.16), in: Capsule())
            }

            Spacer(minLength: 10)

            HStack(alignment: .firstTextBaseline, spacing: 6) {
                Image(systemName: "alarm")
                    .font(.system(size: 16, weight: .medium))
                    .foregroundStyle(stationRed.opacity(0.42))

                Text(wakeTimeText)
                    .font(.system(size: 24, weight: .medium, design: .rounded))
                    .monospacedDigit()
                    .lineLimit(1)
                    .minimumScaleFactor(0.72)
                    .foregroundStyle(stationRed.opacity(0.62))
            }

            Spacer(minLength: 8)

            VStack(alignment: .leading, spacing: 2) {
                Text(timeUntilWakeText)
                    .font(.system(size: 33, weight: .semibold, design: .rounded))
                    .monospacedDigit()
                    .lineLimit(1)
                    .minimumScaleFactor(0.62)
                    .foregroundStyle(stationRed)

                Text("UNTIL MORNING")
                    .font(.system(size: 9, weight: .medium, design: .rounded))
                    .tracking(1.15)
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

    if totalMinutes <= 1 {
        return "SOON"
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
wakeAt: Calendar.current.date(byAdding: .hour, value: 16, to: Date()) ?? Date(),
isArmed: true
)

MorningStationEntry(
    date: Date(),
    wakeAt: Calendar.current.date(byAdding: .hour, value: 8, to: Date()) ?? Date(),
    isArmed: false
)

}
