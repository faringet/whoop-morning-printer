import SwiftUI
import WidgetKit

struct ContentView: View {
@State private var baseURLText = SharedSettingsStore.shared.baseURLText
@State private var displayToken = SharedSettingsStore.shared.displayToken
@State private var isTestingConnection = false
@State private var testMessage: String?
@State private var lastSnapshot: NightDisplaySnapshot?

private var trimmedBaseURLText: String {
    baseURLText.trimmingCharacters(in: .whitespacesAndNewlines)
}

private var trimmedDisplayToken: String {
    displayToken.trimmingCharacters(in: .whitespacesAndNewlines)
}

private var baseURL: URL? {
    URL(string: trimmedBaseURLText)
}

private var canTestConnection: Bool {
    baseURL != nil && !trimmedDisplayToken.isEmpty && !isTestingConnection
}

var body: some View {
    NavigationStack {
        Form {
            Section("Printer Gateway") {
                TextField("Base URL", text: $baseURLText)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .keyboardType(.URL)

                SecureField("Display token", text: $displayToken)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()

                Button {
                    Task {
                        await testConnection()
                    }
                } label: {
                    HStack {
                        if isTestingConnection {
                            ProgressView()
                        }

                        Text(isTestingConnection ? "Testing..." : "Test connection")
                    }
                }
                .disabled(!canTestConnection)
            }

            Section("Widget") {
                Label("Settings are saved for the widget automatically.", systemImage: "checkmark.circle")
                    .foregroundStyle(.green)

                Text("You do not need to keep this app open. After the display token is saved, the StandBy widget can refresh itself from printergateway.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)

                Button {
                    persistSettings()
                    WidgetCenter.shared.reloadTimelines(ofKind: "MorningStationWidget")
                    testMessage = "Widget refresh requested."
                } label: {
                    Label("Refresh widget now", systemImage: "arrow.clockwise")
                }
            }

            if let testMessage {
                Section("Status") {
                    Text(testMessage)
                        .foregroundStyle(statusColor(for: testMessage))
                }
            }

            Section("Last snapshot") {
                if let lastSnapshot {
                    LabeledContent("Status", value: lastSnapshot.status.rawValue)
                    LabeledContent("Timezone", value: lastSnapshot.timezone)
                    LabeledContent("Server time", value: formattedDate(lastSnapshot.serverTime))

                    if let wakePlan = lastSnapshot.wakePlan {
                        LabeledContent("Wake plan ID", value: String(wakePlan.id))
                        LabeledContent("Wake status", value: wakePlan.status)
                        LabeledContent("Wake at", value: formattedDate(wakePlan.wakeAt))
                        LabeledContent("Updated at", value: formattedDate(wakePlan.updatedAt))
                    } else {
                        Text("No wake plan is armed.")
                            .foregroundStyle(.secondary)
                    }
                } else {
                    Text("No snapshot saved yet.")
                        .foregroundStyle(.secondary)
                }
            }
        }
        .navigationTitle("Morning Station")
        .onAppear {
            loadSavedState()
            persistSettings()
        }
        .onChange(of: baseURLText) { _, _ in
            persistSettings()
        }
        .onChange(of: displayToken) { _, _ in
            persistSettings()
        }
    }
}

private func loadSavedState() {
    baseURLText = SharedSettingsStore.shared.baseURLText
    displayToken = SharedSettingsStore.shared.displayToken
    lastSnapshot = SnapshotStore.shared.load()
}

private func persistSettings() {
    SharedSettingsStore.shared.save(
        baseURLText: baseURLText,
        displayToken: displayToken
    )

    WidgetCenter.shared.reloadTimelines(ofKind: "MorningStationWidget")
}

@MainActor
private func testConnection() async {
    guard let baseURL else {
        testMessage = "Invalid base URL."
        return
    }

    guard !trimmedDisplayToken.isEmpty else {
        testMessage = "Display token is empty."
        return
    }

    isTestingConnection = true
    testMessage = nil

    persistSettings()

    do {
        let client = PrinterGatewayClient(
            baseURL: baseURL,
            displayToken: trimmedDisplayToken
        )

        let snapshot = try await client.fetchNightDisplay()

        lastSnapshot = snapshot
        try SnapshotStore.shared.save(snapshot)

        WidgetCenter.shared.reloadTimelines(ofKind: "MorningStationWidget")

        if snapshot.isArmed {
            testMessage = "Connected. Wake plan is armed."
        } else {
            testMessage = "Connected. No wake plan is armed."
        }
    } catch {
        testMessage = error.localizedDescription
    }

    isTestingConnection = false
}

private func statusColor(for message: String) -> Color {
    let lowercasedMessage = message.lowercased()

    if lowercasedMessage.contains("connected") || lowercasedMessage.contains("refresh requested") {
        return .green
    }

    return .red
}

private func formattedDate(_ date: Date) -> String {
    let formatter = DateFormatter()
    formatter.dateFormat = "dd.MM.yyyy HH:mm:ss"
    return formatter.string(from: date)
}

}

#Preview {
ContentView()
}
