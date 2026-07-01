package properties

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"koditon/internal/db"
)

const listingRenovationEventsProjectionVersion = "listing-renovation-events-v1"

func ProjectListingRenovationEvents(ctx context.Context, dbtx db.DBTX, saleListingID uuid.UUID) error {
	_, err := db.New(dbtx).ProjectListingRenovationEvents(ctx, db.ProjectListingRenovationEventsParams{ProjectionVersion: listingRenovationEventsProjectionVersion, SaleListingID: saleListingID})
	if err != nil {
		return fmt.Errorf("project listing renovation events: %w", err)
	}
	return nil
}
