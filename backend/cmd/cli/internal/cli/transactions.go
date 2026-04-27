package cli

import (
	"context"
	"fmt"

	"koditon/internal/sync/prices"
)

type TransactionsFlags struct {
	City   string
	Search string
	Limit  int
}

func RunTransactions(ctx context.Context, svc *prices.Service, f TransactionsFlags) error {
	if f.City == "" {
		return fmt.Errorf("--city is required")
	}
	limit := int32(f.Limit)
	if limit <= 0 {
		limit = 50
	}

	rows, err := svc.SearchTransactionsByCityAndAddress(ctx, f.City, f.Search, limit)
	if err != nil {
		return fmt.Errorf("transactions: %w", err)
	}

	fmt.Println(headerStyle.Render(fmt.Sprintf("%d transactions found", len(rows))))
	fmt.Println()

	if len(rows) == 0 {
		fmt.Println(mutedStyle.Render("No transactions found."))
		return nil
	}

	headers := []string{"Period", "Description", "Type", "Area", "Price", "€/m²", "Postal", "Neighborhood", "Condition"}
	tableRows := make([][]string, 0, len(rows))
	for _, r := range rows {
		tableRows = append(tableRows, []string{
			r.PeriodIdentifier,
			truncate(r.Description, 35),
			r.Type,
			formatAreaFloat(r.Area),
			formatPriceInt(r.Price),
			formatPriceInt(r.PricePerSqm),
			r.PostalCode,
			truncate(r.Neighborhood, 20),
			r.Condition,
		})
	}

	fmt.Print(renderTable(headers, tableRows))

	fmt.Println()
	fmt.Println(mutedStyle.Render(fmt.Sprintf("City: %s", f.City)))
	if f.Search != "" {
		fmt.Println(mutedStyle.Render(fmt.Sprintf("Search: %s", f.Search)))
	}

	return nil
}
