import SwiftUI
import WebKit

struct ContentView: View {
    var body: some View {
        WebView(url: URL(string: "http://localhost:5173/"))
    }
}

#Preview {
    ContentView()
}
