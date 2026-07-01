package properties

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"koditon/internal/db"
)

const listingRenovationEventsProjectionVersion = "listing-renovation-events-v1"

func ProjectListingRenovationEvents(ctx context.Context, dbtx db.DBTX, saleListingID uuid.UUID) error {
	projectionVersion := listingRenovationEventsProjectionVersion
	_, err := db.New(dbtx).ProjectListingRenovationEvents(ctx, db.ProjectListingRenovationEventsParams{ProjectionVersion: &projectionVersion, SaleListingID: &saleListingID})
	if err != nil {
		return fmt.Errorf("project listing renovation events: %w", err)
	}
	return nil
}
