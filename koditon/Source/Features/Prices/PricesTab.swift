import KoditonClient
import SwiftUI

struct PricesTab: View {
    @Environment(AppManager.self) private var appManager
    @State private var showingLocationSheet = false
    @State private var showingAreaSheet = false
    @State private var filter = PricesFilter()
    @State private var transactions: [Components.Schemas.PricesTransaction] = []
    @State private var isLoading = false
    @State private var error: String?
    var body: some View {
        NavigationStack {
            Group {
                if !filter.hasLocationFilter {
                    ContentUnavailableView {
                        Label("No Location Selected", systemImage: "mappin.slash")
                    } description: {
                        Text(
                            "Select a municipality and postal codes to view property transactions.")
                    } actions: {
                        Button("Select Location") {
                            showingLocationSheet = true
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
                        Text("No property transactions found for the selected filters.")
                    } actions: {
                        Button("Change Location") {
                            showingLocationSheet = true
                        }
                        .buttonStyle(.bordered)
                    }
                } else {
                    TransactionListView(transactions: transactions)
                }
            }
            .navigationTitle("Prices")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button {
                        showingLocationSheet = true
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "mappin.circle")
                            if filter.hasLocationFilter {
                                Text(locationSummary)
                                    .lineLimit(1)
                            } else {
                                Text("Location")
                            }
                        }
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        Menu {
                            ForEach(appManager.availability.categories) { category in
                                Button {
                                    toggleCategory(category.value)
                                } label: {
                                    if filter.selectedCategories.contains(category.value) {
                                        Label(category.displayName, systemImage: "checkmark")
                                    } else {
                                        Text(category.displayName)
                                    }
                                }
                            }
                            if !filter.selectedCategories.isEmpty {
                                Divider()
                                Button("Clear Selection", role: .destructive) {
                                    filter.selectedCategories.removeAll()
                                }
                            }
                        } label: {
                            Label(
                                categoryMenuLabel,
                                systemImage: "bed.double"
                            )
                        }
                        Menu {
                            ForEach(appManager.availability.types) { type in
                                Button {
                                    toggleType(type.value)
                                } label: {
                                    if filter.selectedTypes.contains(type.value) {
                                        Label(type.displayName, systemImage: "checkmark")
                                    } else {
                                        Text(type.displayName)
                                    }
                                }
                            }
                            if !filter.selectedTypes.isEmpty {
                                Divider()
                                Button("Clear Selection", role: .destructive) {
                                    filter.selectedTypes.removeAll()
                                }
                            }
                        } label: {
                            Label(
                                typeMenuLabel,
                                systemImage: "building.2"
                            )
                        }
                        Button {
                            showingAreaSheet = true
                        } label: {
                            Label(
                                areaMenuLabel,
                                systemImage: "square.dashed"
                            )
                        }
                        if filter.hasAdditionalFilters {
                            Divider()
                            Button(role: .destructive) {
                                filter.clearAdditionalFilters()
                            } label: {
                                Label("Clear All Filters", systemImage: "xmark.circle")
                            }
                        }
                    } label: {
                        HStack(spacing: 4) {
                            if filter.hasAdditionalFilters {
                                Image(systemName: "line.3.horizontal.decrease.circle.fill")
                            } else {
                                Image(systemName: "line.3.horizontal.decrease.circle")
                            }
                        }
                    }
                }
            }
            .sheet(isPresented: $showingLocationSheet) {
                LocationFilterSheet(
                    selectedMunicipalities: $filter.selectedMunicipalities,
                    selectedPostalCodes: $filter.selectedPostalCodes
                )
            }
            .sheet(isPresented: $showingAreaSheet) {
                AreaFilterSheet(minArea: $filter.minArea, maxArea: $filter.maxArea)
            }
            .onChange(of: filter) { _, _ in
                if filter.hasLocationFilter {
                    Task { await fetchTransactions() }
                }
            }
        }
    }
    private var locationSummary: String {
        let municipalityCount = filter.selectedMunicipalities.count
        let postalCodeCount = filter.selectedPostalCodes.count
        if municipalityCount == 1, let municipalityId = filter.selectedMunicipalities.first {
            let name =
                appManager.availability.municipalities.first { $0.id == municipalityId }?.nameFi
                ?? "1 municipality"
            if postalCodeCount > 0 {
                return "\(name) (\(postalCodeCount))"
            }
            return name
        } else if municipalityCount > 1 {
            return "\(municipalityCount) municipalities"
        }
        return "Location"
    }
    private var categoryMenuLabel: String {
        if filter.selectedCategories.isEmpty {
            return "Room Count"
        }
        return "Room Count (\(filter.selectedCategories.count))"
    }
    private var typeMenuLabel: String {
        if filter.selectedTypes.isEmpty {
            return "Building Type"
        }
        return "Building Type (\(filter.selectedTypes.count))"
    }
    private var areaMenuLabel: String {
        if let min = filter.minArea, let max = filter.maxArea {
            return "Area: \(Int(min))–\(Int(max)) m²"
        } else if let min = filter.minArea {
            return "Area: ≥ \(Int(min)) m²"
        } else if let max = filter.maxArea {
            return "Area: ≤ \(Int(max)) m²"
        }
        return "Area"
    }
    private func toggleCategory(_ categoryValue: String) {
        if filter.selectedCategories.contains(categoryValue) {
            filter.selectedCategories.remove(categoryValue)
        } else {
            filter.selectedCategories.insert(categoryValue)
        }
    }
    private func toggleType(_ typeValue: String) {
        if filter.selectedTypes.contains(typeValue) {
            filter.selectedTypes.remove(typeValue)
        } else {
            filter.selectedTypes.insert(typeValue)
        }
    }
    private func fetchTransactions() async {
        guard filter.hasLocationFilter else { return }
        isLoading = true
        error = nil
        do {
            let params = filter.queryParameters()
            let response = try await appManager.client.api.pricesTransactionsFiltered(
                query: .init(
                    municipalityIds: params.municipalityIds,
                    postalCodeIds: params.postalCodeIds,
                    categories: params.categories,
                    types: params.types,
                    minArea: params.minArea ?? 0,
                    maxArea: params.maxArea ?? 0,
                    limit: 200
                )
            )
            switch response {
            case .ok(let okResponse):
                let body = try okResponse.body.json
                transactions = body.transactions ?? []
            default:
                error = "Failed to load transactions"
            }
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }
}
