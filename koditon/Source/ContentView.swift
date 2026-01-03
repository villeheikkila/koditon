import KoditonClient
import SwiftUI

struct ContentView: View {
    @Environment(AppManager.self) private var appManager
    var body: some View {
        switch appManager.auth.state {
        case .signedOut:
            SignInView()
        case .anonymous(let userId):
            MainView(userId: userId, isAnonymous: true)
        case .authenticated(let userId):
            MainView(userId: userId, isAnonymous: false)
        }
    }
}

struct SignInView: View {
    @Environment(AppManager.self) private var appManager
    @State private var isLoading = false
    @State private var error: String?
    var body: some View {
        VStack(spacing: 24) {
            Spacer()
            Image(systemName: "house.fill")
                .font(.system(size: 80))
                .foregroundStyle(.tint)
            Text("Welcome to Koditon")
                .font(.largeTitle)
                .fontWeight(.bold)
            Spacer()
            if isLoading {
                ProgressView()
            } else {
                VStack(spacing: 16) {
                    SignInWithAppleButtonView()
                        .frame(height: 50)
                    Button {
                        Task { await signInAnonymously() }
                    } label: {
                        Text("Continue as Guest")
                            .frame(maxWidth: .infinity)
                            .frame(height: 50)
                    }
                    .buttonStyle(.bordered)
                }
                .padding(.horizontal, 32)
            }
            if let error {
                Text(error)
                    .foregroundStyle(.red)
                    .font(.caption)
            }
            Spacer()
        }
    }
    private func signInAnonymously() async {
        isLoading = true
        error = nil
        do {
            try await appManager.auth.signInAnonymously()
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }
}

struct MainView: View {
    @Environment(AppManager.self) private var appManager
    let userId: String
    let isAnonymous: Bool
    var body: some View {
        TabView {
            Tab("Home", systemImage: "house") {
                HomeTab(userId: userId, isAnonymous: isAnonymous)
            }
            Tab("Prices", systemImage: "chart.line.uptrend.xyaxis") {
                PricesTab()
            }
        }
    }
}

struct HomeTab: View {
    @Environment(AppManager.self) private var appManager
    let userId: String
    let isAnonymous: Bool
    var body: some View {
        NavigationStack {
            VStack(spacing: 16) {
                Image(systemName: "checkmark.circle.fill")
                    .font(.system(size: 60))
                    .foregroundStyle(.green)
                Text("Signed In")
                    .font(.title)
                Text(isAnonymous ? "Anonymous User" : "Apple User")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                Text(userId)
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .monospaced()
            }
            .navigationTitle("Koditon")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Sign Out") {
                        Task { await appManager.auth.signOut() }
                    }
                }
            }
        }
    }
}
