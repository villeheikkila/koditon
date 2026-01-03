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

struct PricesTab: View {
    @Environment(AppManager.self) private var appManager
    @State private var showingLocationSelector = false
    @State private var selectedCity: Components.Schemas.PostalCity?
    @State private var selectedPostalCode: Components.Schemas.PostalCode?
    @State private var transactions: [Components.Schemas.PricesTransaction] = []
    @State private var isLoading = false
    @State private var error: String?
    var body: some View {
        NavigationStack {
            Group {
                if selectedCity == nil || selectedPostalCode == nil {
                    ContentUnavailableView {
                        Label("No Location Selected", systemImage: "mappin.slash")
                    } description: {
                        Text("Select a city and postal code to view property transactions.")
                    } actions: {
                        Button("Select Location") {
                            showingLocationSelector = true
                        }
                        .buttonStyle(.borderedProminent)
                    }
                } else if isLoading {
                    ProgressView("Loading transactions...")
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") {
                            Task { await fetchTransactions() }
                        }
                        .buttonStyle(.bordered)
                    }
                } else if transactions.isEmpty {
                    ContentUnavailableView {
                        Label("No Transactions", systemImage: "doc.text.magnifyingglass")
                    } description: {
                        Text("No property transactions found for this area.")
                    } actions: {
                        Button("Change Location") {
                            showingLocationSelector = true
                        }
                        .buttonStyle(.bordered)
                    }
                } else {
                    TransactionListView(transactions: transactions)
                }
            }
            .navigationTitle("Prices")
            .toolbar {
                if selectedCity != nil {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button {
                            showingLocationSelector = true
                        } label: {
                            HStack(spacing: 4) {
                                Text(selectedCity?.nameFi ?? "")
                                if let postalCode = selectedPostalCode {
                                    Text(postalCode.code)
                                        .foregroundStyle(.secondary)
                                }
                                Image(systemName: "chevron.down")
                                    .font(.caption)
                            }
                        }
                    }
                }
            }
            .sheet(isPresented: $showingLocationSelector) {
                LocationSelectorSheet(
                    selectedCity: $selectedCity,
                    selectedPostalCode: $selectedPostalCode
                )
            }
            .onChange(of: selectedPostalCode) { _, newValue in
                if newValue != nil {
                    Task { await fetchTransactions() }
                }
            }
        }
    }

    private func fetchTransactions() async {
        guard let city = selectedCity, let postalCode = selectedPostalCode else {
            print(
                "❌ Guard failed: city=\(selectedCity?.id ?? "nil"), postalCode=\(selectedPostalCode?.id ?? "nil")"
            )
            return
        }
        print("🔄 Starting fetch: municipalityId=\(city.id), postalCodeId=\(postalCode.id)")
        isLoading = true
        error = nil
        do {
            print("📡 Calling API...")
            let response = try await appManager.client.api.pricesTransactions(
                query: .init(municipalityId: city.id, postalCodeId: postalCode.id)
            )
            print("📥 Response received")
            switch response {
            case .ok(let okResponse):
                print("✅ OK response")
                let body = try okResponse.body.json
                transactions = body.transactions ?? []
                print("📊 Loaded \(transactions.count) transactions")
            default:
                print("⚠️ Non-OK response: \(response)")
                error = "Failed to load transactions"
            }
        } catch {
            print("❌ Error: \(error)")
            self.error = error.localizedDescription
        }
        print("✅ Setting isLoading = false")
        isLoading = false
    }
}

struct LocationSelectorSheet: View {
    @Environment(AppManager.self) private var appManager
    @Environment(\.dismiss) private var dismiss
    @Binding var selectedCity: Components.Schemas.PostalCity?
    @Binding var selectedPostalCode: Components.Schemas.PostalCode?
    @State private var cities: [Components.Schemas.PostalCity] = []
    @State private var isLoading = true
    @State private var error: String?
    @State private var searchText = ""
    @State private var tempSelectedCity: Components.Schemas.PostalCity?
    @State private var tempSelectedPostalCode: Components.Schemas.PostalCode?
    private var filteredCities: [Components.Schemas.PostalCity] {
        guard !searchText.isEmpty else { return cities }
        return cities.filter { city in
            city.nameFi.localizedCaseInsensitiveContains(searchText)
                || city.code.localizedCaseInsensitiveContains(searchText)
        }
    }

    private var filteredPostalCodes: [Components.Schemas.PostalCode] {
        guard let city = tempSelectedCity, let postalCodes = city.postalCodes else { return [] }
        guard !searchText.isEmpty else { return postalCodes }
        return postalCodes.filter { postalCode in
            postalCode.nameFi.localizedCaseInsensitiveContains(searchText)
                || postalCode.code.localizedCaseInsensitiveContains(searchText)
        }
    }

    var body: some View {
        NavigationStack {
            Group {
                if isLoading {
                    ProgressView("Loading cities...")
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") {
                            Task { await fetchCities() }
                        }
                        .buttonStyle(.bordered)
                    }
                } else if tempSelectedCity == nil {
                    List(filteredCities, id: \.id) { city in
                        Button {
                            tempSelectedCity = city
                            searchText = ""
                        } label: {
                            HStack {
                                VStack(alignment: .leading) {
                                    Text(city.nameFi)
                                    Text(city.code)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                if let count = city.postalCodes?.count {
                                    Text("\(count) postal codes")
                                        .font(.caption)
                                        .foregroundStyle(.tertiary)
                                }
                                Image(systemName: "chevron.right")
                                    .font(.caption)
                                    .foregroundStyle(.tertiary)
                            }
                        }
                        .foregroundStyle(.primary)
                    }
                    .searchable(text: $searchText, prompt: "Search cities")
                } else {
                    List(filteredPostalCodes, id: \.id) { postalCode in
                        Button {
                            tempSelectedPostalCode = postalCode
                            selectedCity = tempSelectedCity
                            selectedPostalCode = tempSelectedPostalCode
                            dismiss()
                        } label: {
                            HStack {
                                VStack(alignment: .leading) {
                                    Text(postalCode.nameFi)
                                    Text(postalCode.code)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                if selectedCity?.id == tempSelectedCity?.id
                                    && selectedPostalCode?.id == postalCode.id
                                {
                                    Image(systemName: "checkmark")
                                        .foregroundStyle(.tint)
                                }
                            }
                        }
                        .foregroundStyle(.primary)
                    }
                    .searchable(text: $searchText, prompt: "Search postal codes")
                }
            }
            .navigationTitle(tempSelectedCity?.nameFi ?? "Select City")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    if tempSelectedCity != nil {
                        Button("Back") {
                            tempSelectedCity = nil
                            tempSelectedPostalCode = nil
                            searchText = ""
                        }
                    } else {
                        Button("Cancel") {
                            dismiss()
                        }
                    }
                }
            }
        }
        .task {
            await fetchCities()
        }
        .onAppear {
            tempSelectedCity = selectedCity
            tempSelectedPostalCode = selectedPostalCode
        }
    }

    private func fetchCities() async {
        isLoading = true
        error = nil
        do {
            let response = try await appManager.client.api.postalCities()
            switch response {
            case .ok(let okResponse):
                let body = try okResponse.body.json
                cities = body.cities ?? []
            default:
                error = "Failed to load cities"
            }
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }
}

struct TransactionListView: View {
    let transactions: [Components.Schemas.PricesTransaction]
    var body: some View {
        List(transactions, id: \.id) { transaction in
            TransactionRow(transaction: transaction)
        }
    }
}

struct TransactionRow: View {
    let transaction: Components.Schemas.PricesTransaction
    private var formattedPrice: String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .currency
        formatter.currencyCode = "EUR"
        formatter.maximumFractionDigits = 0
        return formatter.string(from: NSNumber(value: transaction.price))
            ?? "\(transaction.price) EUR"
    }

    private var formattedPricePerSqm: String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .currency
        formatter.currencyCode = "EUR"
        formatter.maximumFractionDigits = 0
        return
            "\(formatter.string(from: NSNumber(value: transaction.pricePerSquareMeter)) ?? "\(transaction.pricePerSquareMeter)")/m²"
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(transaction.description)
                    .font(.headline)
                Spacer()
                Text(formattedPrice)
                    .font(.subheadline)
                    .fontWeight(.semibold)
            }
            HStack {
                Label(
                    "\(String(format: "%.1f", transaction.area)) m²", systemImage: "square.dashed"
                )
                Spacer()
                Text(formattedPricePerSqm)
                    .foregroundStyle(.secondary)
            }
            .font(.subheadline)
            HStack {
                if transaction.buildYear > 0 {
                    Label("\(String(transaction.buildYear))", systemImage: "calendar")
                }
                if let condition = transaction.condition, !condition.isEmpty {
                    Text(condition)
                }
                Spacer()
                Text(transaction.category)
                    .font(.caption)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2)
                    .background(.quaternary)
                    .clipShape(Capsule())
            }
            .font(.caption)
            .foregroundStyle(.secondary)
        }
        .padding(.vertical, 4)
    }
}

#Preview {
    ContentView()
}
