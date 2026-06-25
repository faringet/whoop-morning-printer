import SwiftUI
import WidgetKit

struct ContentView: View {
    @AppStorage("printerGatewayBaseURL")
    private var baseURLText: String = "https://printergateway.veloranet.ru"

    @AppStorage("printerGatewayDisplayToken")
    private var displayToken: String = ""

    @State private var isTestingConnection = false
    @State private var testMessage = "Not tested yet"
    @State private var lastSnapshot: NightDisplaySnapshot?

    var body: some View {
        NavigationStack {
            Form {
                Section("Printergateway") {
                    TextField("Base URL", text: $baseURLText)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()

                    SecureField("Display token", text: $displayToken)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }

                Section("Connection") {
                    Button {
                        Task {
                            await testConnection()
                        }
                    } label: {
                        if isTestingConnection {
                            ProgressView()
                        } else {
                            Text("Test connection")
                        }
                    }
                    .disabled(isTestingConnection)

                    Text(testMessage)
                        .font(.footnote)
                        .foregroundColor(lastSnapshot == nil ? .secondary : .green)
                }

                if let lastSnapshot {
                    Section("Night display snapshot") {
                        LabeledContent("Status", value: lastSnapshot.status.rawValue)

                        LabeledContent("Timezone", value: lastSnapshot.timezone)

                        LabeledContent(
                            "Server time",
                            value: formatDate(lastSnapshot.serverTime)
                        )

                        if let wakePlan = lastSnapshot.wakePlan {
                            LabeledContent("Wake plan ID", value: "\(wakePlan.id)")
                            LabeledContent("Wake status", value: wakePlan.status)
                            LabeledContent("Wake at", value: formatDate(wakePlan.wakeAt))
                        } else {
                            LabeledContent("Wake plan", value: "Not armed")
                        }
                    }
                }

                Section("Widget") {
                    Text("The StandBy widget is currently using mock data. Real data connection will be added after this gateway test works.")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
            }
            .navigationTitle("Morning Station")
        }
    }

    private func testConnection() async {
        isTestingConnection = true
        testMessage = "Testing..."
        lastSnapshot = nil

        defer {
            isTestingConnection = false
        }

        guard let baseURL = URL(string: baseURLText.trimmingCharacters(in: .whitespacesAndNewlines)) else {
            testMessage = "Invalid base URL"
            return
        }

        do {
            let client = PrinterGatewayClient(
                baseURL: baseURL,
                displayToken: displayToken
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
    }

    private func formatDate(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        formatter.timeStyle = .short

        return formatter.string(from: date)
    }
}

#Preview {
    ContentView()
}
