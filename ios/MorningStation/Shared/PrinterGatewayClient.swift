import Foundation

enum PrinterGatewayClientError: LocalizedError, Equatable {
    case invalidBaseURL
    case missingToken
    case invalidResponse
    case httpStatus(Int)
    case emptyResponse

    var errorDescription: String? {
        switch self {
        case .invalidBaseURL:
            return "Invalid gateway URL."
        case .missingToken:
            return "Display token is empty."
        case .invalidResponse:
            return "Invalid response from printergateway."
        case .httpStatus(let statusCode):
            return "Printergateway returned HTTP \(statusCode)."
        case .emptyResponse:
            return "Printergateway returned an empty response."
        }
    }
}

struct PrinterGatewayClient {
    let baseURL: URL
    let displayToken: String

    func fetchNightDisplay() async throws -> NightDisplaySnapshot {
        let trimmedToken = displayToken.trimmingCharacters(in: .whitespacesAndNewlines)

        guard !trimmedToken.isEmpty else {
            throw PrinterGatewayClientError.missingToken
        }

        guard let endpointURL = makeEndpointURL(path: "/v1/night-display") else {
            throw PrinterGatewayClientError.invalidBaseURL
        }

        var request = URLRequest(url: endpointURL)
        request.httpMethod = "GET"
        request.timeoutInterval = 12
        request.setValue("Bearer \(trimmedToken)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Accept")

        let (data, response) = try await URLSession.shared.data(for: request)

        guard !data.isEmpty else {
            throw PrinterGatewayClientError.emptyResponse
        }

        guard let httpResponse = response as? HTTPURLResponse else {
            throw PrinterGatewayClientError.invalidResponse
        }

        guard (200..<300).contains(httpResponse.statusCode) else {
            throw PrinterGatewayClientError.httpStatus(httpResponse.statusCode)
        }

        return try JSONDecoder().decode(NightDisplaySnapshot.self, from: data)
    }

    private func makeEndpointURL(path: String) -> URL? {
        let normalizedBaseURL = baseURL.absoluteString.trimmingCharacters(
            in: CharacterSet(charactersIn: "/")
        )

        return URL(string: normalizedBaseURL + path)
    }
}
