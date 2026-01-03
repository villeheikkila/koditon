import KoditonClient
import SwiftUI

struct LocationFilterSheet: View {
    @Environment(AppManager.self) private var appManager
    @Environment(\.dismiss) private var dismiss
    @Binding var selectedMunicipalities: Set<String>
    @Binding var selectedPostalCodes: Set<String>
    @State private var tempMunicipalities: Set<String> = []
    @State private var tempPostalCodes: Set<String> = []
    @State private var searchText = ""
    var body: some View {
        NavigationStack {
            List {
                ForEach(filteredMunicipalities) { municipality in
                    NavigationLink {
                        PostalCodeSelectionView(
                            municipality: municipality,
                            selectedMunicipalities: $tempMunicipalities,
                            selectedPostalCodes: $tempPostalCodes
                        )
                    } label: {
                        MunicipalityRow(
                            municipality: municipality,
                            isSelected: tempMunicipalities.contains(municipality.id),
                            selectedPostalCodeCount: selectedPostalCodeCount(for: municipality.id),
                            totalPostalCodeCount: appManager.availability.postalCodes(
                                for: municipality.id
                            ).count
                        )
                    }
                }
            }
            .listStyle(.plain)
            .searchable(text: $searchText, prompt: "Search municipalities")
            .navigationTitle("Select Location")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        dismiss()
                    }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Apply") {
                        selectedMunicipalities = tempMunicipalities
                        selectedPostalCodes = tempPostalCodes
                        dismiss()
                    }
                    .disabled(tempMunicipalities.isEmpty)
                }
                ToolbarItem(placement: .bottomBar) {
                    Button("Clear All", role: .destructive) {
                        tempMunicipalities.removeAll()
                        tempPostalCodes.removeAll()
                    }
                    .disabled(tempMunicipalities.isEmpty && tempPostalCodes.isEmpty)
                }
            }
        }
        .onAppear {
            tempMunicipalities = selectedMunicipalities
            tempPostalCodes = selectedPostalCodes
        }
    }
    private var filteredMunicipalities: [AvailableMunicipality] {
        let municipalities = appManager.availability.municipalities
        guard !searchText.isEmpty else { return municipalities }
        return municipalities.filter { m in
            m.nameFi.localizedCaseInsensitiveContains(searchText)
                || m.code.localizedCaseInsensitiveContains(searchText)
                || (m.nameSv?.localizedCaseInsensitiveContains(searchText) ?? false)
        }
    }
    private func selectedPostalCodeCount(for municipalityId: String) -> Int {
        appManager.availability.postalCodes(for: municipalityId)
            .filter { tempPostalCodes.contains($0.id) }
            .count
    }
}

private struct MunicipalityRow: View {
    let municipality: AvailableMunicipality
    let isSelected: Bool
    let selectedPostalCodeCount: Int
    let totalPostalCodeCount: Int
    var body: some View {
        HStack {
            VStack(alignment: .leading) {
                Text(municipality.nameFi)
                if isSelected {
                    if selectedPostalCodeCount > 0 {
                        Text(
                            "\(selectedPostalCodeCount) of \(totalPostalCodeCount) postal codes selected"
                        )
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    } else {
                        Text("All \(totalPostalCodeCount) postal codes")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                } else {
                    Text("\(totalPostalCodeCount) postal codes")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            Spacer()
            if isSelected {
                Image(systemName: "checkmark.circle.fill")
                    .foregroundStyle(.tint)
            }
        }
    }
}

private struct PostalCodeSelectionView: View {
    @Environment(AppManager.self) private var appManager
    let municipality: AvailableMunicipality
    @Binding var selectedMunicipalities: Set<String>
    @Binding var selectedPostalCodes: Set<String>
    @State private var searchText = ""
    private var postalCodes: [AvailablePostalCode] {
        appManager.availability.postalCodes(for: municipality.id)
    }
    private var filteredPostalCodes: [AvailablePostalCode] {
        guard !searchText.isEmpty else { return postalCodes }
        return postalCodes.filter { pc in
            pc.nameFi.localizedCaseInsensitiveContains(searchText)
                || pc.code.localizedCaseInsensitiveContains(searchText)
                || (pc.nameSv?.localizedCaseInsensitiveContains(searchText) ?? false)
        }
    }
    private var isMunicipalitySelected: Bool {
        selectedMunicipalities.contains(municipality.id)
    }
    private var selectedPostalCodesForMunicipality: Set<String> {
        Set(postalCodes.map(\.id).filter { selectedPostalCodes.contains($0) })
    }
    private var hasAllPostalCodesSelected: Bool {
        let allIds = Set(postalCodes.map(\.id))
        return allIds.allSatisfy { selectedPostalCodes.contains($0) }
    }
    private var hasNoSpecificPostalCodesSelected: Bool {
        selectedPostalCodesForMunicipality.isEmpty
    }
    var body: some View {
        List {
            Section {
                Button {
                    selectAllPostalCodes()
                } label: {
                    HStack {
                        VStack(alignment: .leading) {
                            Text("Select All")
                                .fontWeight(.medium)
                            Text("Include all \(postalCodes.count) postal codes")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        if isMunicipalitySelected && hasNoSpecificPostalCodesSelected {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundStyle(.tint)
                        } else if hasAllPostalCodesSelected {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundStyle(.tint)
                        }
                    }
                }
                .foregroundStyle(.primary)
            }
            Section {
                ForEach(filteredPostalCodes) { postalCode in
                    Button {
                        togglePostalCode(postalCode)
                    } label: {
                        HStack {
                            VStack(alignment: .leading) {
                                Text(postalCode.nameFi)
                                Text(postalCode.code)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            if selectedPostalCodes.contains(postalCode.id) {
                                Image(systemName: "checkmark")
                                    .foregroundStyle(.tint)
                            }
                        }
                    }
                    .foregroundStyle(.primary)
                }
            } header: {
                Text("Postal Codes")
            }
        }
        .listStyle(.insetGrouped)
        .searchable(text: $searchText, prompt: "Search postal codes")
        .navigationTitle(municipality.nameFi)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .bottomBar) {
                if isMunicipalitySelected {
                    Button("Deselect Municipality", role: .destructive) {
                        deselectMunicipality()
                    }
                }
            }
        }
    }
    private func selectAllPostalCodes() {
        selectedMunicipalities.insert(municipality.id)
        for pc in postalCodes {
            selectedPostalCodes.remove(pc.id)
        }
    }
    private func togglePostalCode(_ postalCode: AvailablePostalCode) {
        if selectedPostalCodes.contains(postalCode.id) {
            selectedPostalCodes.remove(postalCode.id)
            if selectedPostalCodesForMunicipality.isEmpty
                || (selectedPostalCodesForMunicipality.count == 1
                    && selectedPostalCodesForMunicipality.contains(postalCode.id))
            {
                if !hasNoSpecificPostalCodesSelected
                    || selectedPostalCodesForMunicipality.count <= 1
                {
                    let remaining = selectedPostalCodesForMunicipality.subtracting([postalCode.id])
                    if remaining.isEmpty {
                        selectedMunicipalities.remove(municipality.id)
                    }
                }
            }
        } else {
            selectedMunicipalities.insert(municipality.id)
            selectedPostalCodes.insert(postalCode.id)
        }
    }
    private func deselectMunicipality() {
        selectedMunicipalities.remove(municipality.id)
        for pc in postalCodes {
            selectedPostalCodes.remove(pc.id)
        }
    }
}
