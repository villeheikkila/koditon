import KoditonClient
import SwiftUI

struct PricesFilter: Equatable, Sendable {
    var selectedMunicipalities: Set<String> = []
    var selectedPostalCodes: Set<String> = []
    var selectedCategories: Set<String> = []
    var selectedTypes: Set<String> = []
    var minArea: Double?
    var maxArea: Double?
    var isEmpty: Bool {
        selectedMunicipalities.isEmpty
            && selectedPostalCodes.isEmpty
            && selectedCategories.isEmpty
            && selectedTypes.isEmpty
            && minArea == nil
            && maxArea == nil
    }
    var hasLocationFilter: Bool {
        !selectedMunicipalities.isEmpty || !selectedPostalCodes.isEmpty
    }
    var hasAdditionalFilters: Bool {
        !selectedCategories.isEmpty || !selectedTypes.isEmpty || minArea != nil || maxArea != nil
    }
    mutating func clear() {
        selectedMunicipalities.removeAll()
        selectedPostalCodes.removeAll()
        selectedCategories.removeAll()
        selectedTypes.removeAll()
        minArea = nil
        maxArea = nil
    }
    mutating func clearAdditionalFilters() {
        selectedCategories.removeAll()
        selectedTypes.removeAll()
        minArea = nil
        maxArea = nil
    }
    func queryParameters() -> (
        municipalityIds: String?,
        postalCodeIds: String?,
        categories: String?,
        types: String?,
        minArea: Double?,
        maxArea: Double?
    ) {
        let municipalityIds =
            selectedMunicipalities.isEmpty
            ? nil : selectedMunicipalities.sorted().joined(separator: ",")
        let postalCodeIds =
            selectedPostalCodes.isEmpty ? nil : selectedPostalCodes.sorted().joined(separator: ",")
        let categories =
            selectedCategories.isEmpty ? nil : selectedCategories.sorted().joined(separator: ",")
        let types = selectedTypes.isEmpty ? nil : selectedTypes.sorted().joined(separator: ",")
        return (municipalityIds, postalCodeIds, categories, types, minArea, maxArea)
    }
}

struct AreaFilterSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Binding var minArea: Double?
    @Binding var maxArea: Double?
    @State private var tempMinArea: Double?
    @State private var tempMaxArea: Double?
    var body: some View {
        NavigationStack {
            Form {
                Section("Living Area (m²)") {
                    HStack {
                        Text("Min")
                            .foregroundStyle(.secondary)
                        TextField("Min", value: $tempMinArea, format: .number)
                            .keyboardType(.decimalPad)
                            .textFieldStyle(.roundedBorder)
                    }
                    HStack {
                        Text("Max")
                            .foregroundStyle(.secondary)
                        TextField("Max", value: $tempMaxArea, format: .number)
                            .keyboardType(.decimalPad)
                            .textFieldStyle(.roundedBorder)
                    }
                }
                Section {
                    presetButtons
                }
            }
            .navigationTitle("Area")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        dismiss()
                    }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Apply") {
                        minArea = tempMinArea
                        maxArea = tempMaxArea
                        dismiss()
                    }
                }
                ToolbarItem(placement: .bottomBar) {
                    Button("Clear", role: .destructive) {
                        tempMinArea = nil
                        tempMaxArea = nil
                    }
                    .disabled(tempMinArea == nil && tempMaxArea == nil)
                }
            }
        }
        .onAppear {
            tempMinArea = minArea
            tempMaxArea = maxArea
        }
    }
    @ViewBuilder
    private var presetButtons: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Quick Presets")
                .font(.caption)
                .foregroundStyle(.secondary)
            HStack {
                presetButton("< 30 m²", min: nil, max: 30)
                presetButton("30–50 m²", min: 30, max: 50)
                presetButton("50–80 m²", min: 50, max: 80)
            }
            HStack {
                presetButton("80–120 m²", min: 80, max: 120)
                presetButton("> 120 m²", min: 120, max: nil)
            }
        }
    }
    private func presetButton(_ title: String, min: Double?, max: Double?) -> some View {
        Button {
            tempMinArea = min
            tempMaxArea = max
        } label: {
            Text(title)
                .font(.caption)
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
                .background(
                    tempMinArea == min && tempMaxArea == max
                        ? Color.accentColor : Color.secondary.opacity(0.2)
                )
                .foregroundStyle(tempMinArea == min && tempMaxArea == max ? .white : .primary)
                .clipShape(Capsule())
        }
        .buttonStyle(.plain)
    }
}
