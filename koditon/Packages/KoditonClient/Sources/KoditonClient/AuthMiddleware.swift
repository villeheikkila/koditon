import Foundation
import HTTPTypes
import OpenAPIRuntime

public struct AuthMiddleware: ClientMiddleware {
    private let tokenProvider: @Sendable () async throws -> String?
    private let deviceID: String
    public init(tokenProvider: @escaping @Sendable () async throws -> String?, deviceID: String) {
        self.tokenProvider = tokenProvider
        self.deviceID = deviceID
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
        return try await next(request, body, baseURL)
    }
}
