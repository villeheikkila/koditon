import Foundation
@_exported import HTTPTypes
@_exported import OpenAPIRuntime
import OpenAPIURLSession

public struct KoditonClient: Sendable {
    private let client: Client

    public init(baseURL: URL, middlewares: [any ClientMiddleware] = []) {
        self.client = Client(
            serverURL: baseURL,
            configuration: .init(multipartBoundaryGenerator: .random),
            transport: URLSessionTransport(),
            middlewares: middlewares
        )
    }
    public var api: Client { client }
}
