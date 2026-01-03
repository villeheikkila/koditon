import KoditonClient
import SwiftUI

struct TransactionListView: View {
    let transactions: [Components.Schemas.PricesTransaction]
    var body: some View {
        ScrollView {
            LazyVStack(spacing: 16) {
                ForEach(transactions, id: \.id) { transaction in
                    TransactionCard(transaction: transaction)
                }
            }
            .padding(.horizontal)
            .padding(.vertical, 8)
        }
        .background(Color(.systemGroupedBackground))
    }
}

struct TransactionCard: View {
    @Environment(AppManager.self) private var appManager
    let transaction: Components.Schemas.PricesTransaction
    private var formattedPrice: String {
        transaction.price.formatted(.currency(code: "EUR").precision(.fractionLength(0)))
    }
    private var formattedPricePerSqm: String {
        "\(transaction.pricePerSquareMeter.formatted(.currency(code: "EUR").precision(.fractionLength(0))))/m²"
    }
    private var categoryDisplayName: String {
        appManager.availability.categoryDisplayName(for: transaction.category)
    }
    private var typeDisplayName: String {
        appManager.availability.typeDisplayName(for: transaction._type)
    }
    private var formattedDate: String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: transaction.createdAt) {
            return date.formatted(.dateTime.day().month(.abbreviated).year())
        }
        formatter
            .formatOptions = [.withInternetDateTime]
        if let date = formatter.date(from: transaction.createdAt) {
            return date.formatted(.dateTime.day().month(.abbreviated).year())
        }
        return transaction.createdAt
    }
    private var locationText: String {
        "\(transaction.postalCodeCode) \(transaction.postalCodeNameFi)"
    }
    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Top section: Date
            HStack {
                Spacer()
                Text(formattedDate)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            .padding(.bottom, 12)
            // Main content
            HStack(alignment: .top, spacing: 16) {
                // Left column: Property info
                VStack(alignment: .leading, spacing: 8) {
                    // Description
                    Text(transaction.description)
                        .font(.title3)
                        .fontWeight(.semibold)
                        .lineLimit(2)
                    // Location
                    VStack(alignment: .leading, spacing: 2) {
                        Text(locationText)
                            .font(.subheadline)
                            .foregroundStyle(.primary)
                        Text(transaction.municipalityNameFi)
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                }
                Spacer()
                // Right column: Price
                VStack(alignment: .trailing, spacing: 4) {
                    Text(formattedPrice)
                        .font(.title2)
                        .fontWeight(.bold)
                    Text(formattedPricePerSqm)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            .padding(.bottom, 16)
            // Divider
            Rectangle()
                .fill(Color(.separator).opacity(0.3))
                .frame(height: 1)
                .padding(.bottom, 16)
            // Bottom section: Details grid
            HStack(spacing: 24) {
                // Area
                DetailItem(
                    icon: "square.split.bottomrightquarter",
                    value: "\(transaction.area.formatted(.number.precision(.fractionLength(1)))) m²"
                )
                // Build year
                if transaction.buildYear > 0 {
                    DetailItem(
                        icon: "building.2",
                        value: String(transaction.buildYear)
                    )
                }
                // Condition
                if let condition = transaction.condition, !condition.isEmpty {
                    DetailItem(
                        icon: "star",
                        value: condition
                    )
                }
                Spacer()
            }
            .padding(.bottom, 12)
            // Tags
            HStack(spacing: 8) {
                TagView(text: typeDisplayName, style: .primary)
                TagView(text: categoryDisplayName, style: .secondary)
                Spacer()
            }
        }
        .padding(16)
        .background(Color(.systemBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .shadow(color: .black.opacity(0.06), radius: 8, x: 0, y: 2)
    }
}

struct DetailItem: View {
    let icon: String
    let value: String
    var body: some View {
        HStack(spacing: 6) {
            Image(systemName: icon)
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.subheadline)
                .foregroundStyle(.primary)
        }
    }
}

struct TagView: View {
    enum Style {
        case primary
        case secondary
    }
    let text: String
    let style: Style
    var body: some View {
        Text(text)
            .font(.caption)
            .fontWeight(.medium)
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(backgroundColor)
            .foregroundStyle(foregroundColor)
            .clipShape(RoundedRectangle(cornerRadius: 6))
    }
    private var backgroundColor: Color {
        switch style {
        case .primary:
            return .blue.opacity(0.12)
        case .secondary:
            return Color(.systemGray5)
        }
    }
    private var foregroundColor: Color {
        switch style {
        case .primary:
            return .blue
        case .secondary:
            return .secondary
        }
    }
}
