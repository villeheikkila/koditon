import Foundation
import HTTPTypes
import OpenAPIRuntime

public struct AuthMiddleware: ClientMiddleware {
    private let tokenProvider: @Sendable () async throws -> String?
    private let onUnauthorized: (@Sendable () async throws -> String?)?
    private let deviceID: String
    public init(
        tokenProvider: @escaping @Sendable () async throws -> String?,
        deviceID: String,
        onUnauthorized: (@Sendable () async throws -> String?)? = nil
    ) {
        self.tokenProvider = tokenProvider
        self.deviceID = deviceID
        self.onUnauthorized = onUnauthorized
    }
    public func intercept(
        _ request: HTTPRequest,
        body: HTTPBody?,
        baseURL: URL,
        operationID: String,
        next: @Sendable (HTTPRequest, HTTPBody?, URL) async throws -> (HTTPResponse, HTTPBody?)
    ) async throws -> (HTTPResponse, HTTPBody?) {
        var request = request
        request.headerFields[.init("X-Device-ID")!] = deviceID
        if let token = try? await tokenProvider() {
            request.headerFields[.authorization] = "Bearer \(token)"
        }
        let (response, responseBody) = try await next(request, body, baseURL)
        if response.status == .unauthorized, let onUnauthorized {
            do {
                if let newToken = try await onUnauthorized() {
                    var retryRequest = request
                    retryRequest.headerFields[.authorization] = "Bearer \(newToken)"
                    return try await next(retryRequest, body, baseURL)
                }
            } catch {
                return (response, responseBody)
            }
        }
        return (response, responseBody)
    }
}
