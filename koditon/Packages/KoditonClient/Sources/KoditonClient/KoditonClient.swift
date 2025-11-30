import Foundation
import OpenAPIRuntime
import OpenAPIURLSession

@main
struct KoditonClient {
    static func main() {
        let client = try Client(
            serverURL: URL(string: "http://localhost:8080")!,
            transport: URLSessionTransport()
        )
    }
}
