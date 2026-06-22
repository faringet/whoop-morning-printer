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

        let nextUpdate = Calendar.current.date(byAdding: .minute, value: 15, to: now) ?? now.addingTimeInterval(900)
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
        VStack(spacing: 8) {
            Text("WAKE")
                .font(.system(size: 13, weight: .semibold, design: .rounded))
                .tracking(1.8)
                .foregroundStyle(.red.opacity(0.55))

            Text(wakeTimeText)
                .font(.system(size: 42, weight: .semibold, design: .rounded))
                .monospacedDigit()
                .foregroundStyle(.red)

            Spacer(minLength: 2)

            Text(timeUntilWakeText)
                .font(.system(size: 24, weight: .semibold, design: .rounded))
                .monospacedDigit()
                .foregroundStyle(.red.opacity(0.9))

            Text("UNTIL MORNING")
                .font(.system(size: 10, weight: .medium, design: .rounded))
                .tracking(1.3)
                .foregroundStyle(.red.opacity(0.45))

            Spacer(minLength: 2)

            Text("ARMED")
                .font(.system(size: 11, weight: .semibold, design: .rounded))
                .tracking(1.6)
                .foregroundStyle(.red.opacity(0.38))
        }
        .padding(16)
    }

    private var notArmedView: some View {
        VStack(spacing: 10) {
            Text("MORNING")
                .font(.system(size: 13, weight: .semibold, design: .rounded))
                .tracking(1.8)
                .foregroundStyle(.red.opacity(0.45))

            Spacer(minLength: 4)

            Text("NOT\nARMED")
                .font(.system(size: 26, weight: .semibold, design: .rounded))
                .multilineTextAlignment(.center)
                .foregroundStyle(.red.opacity(0.85))

            Spacer(minLength: 4)

            Text("SET WAKE\nIN TELEGRAM")
                .font(.system(size: 10, weight: .medium, design: .rounded))
                .tracking(1.1)
                .multilineTextAlignment(.center)
                .foregroundStyle(.red.opacity(0.4))
        }
        .padding(16)
    }

    private var wakeTimeText: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: entry.wakeAt)
    }

    private var timeUntilWakeText: String {
        let seconds = max(0, Int(entry.wakeAt.timeIntervalSince(Date())))
        let hours = seconds / 3600
        let minutes = (seconds % 3600) / 60

        if hours > 0 {
            return "\(hours)H \(minutes)M"
        }

        return "\(minutes)M"
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
}
