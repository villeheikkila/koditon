package cli

import (
	"context"
	"fmt"
	"strings"

	"koditon/internal/domain/ads"
)

func RunDetail(ctx context.Context, svc *ads.Service, input, shortcutBase, frontdoorBase, webBaseURL string) error {
	canonicalID, err := ads.ResolveInput(input, shortcutBase, frontdoorBase)
	if err != nil {
		return fmt.Errorf("resolve input: %w", err)
	}

	detail, err := svc.DetailByCanonicalID(ctx, canonicalID)
	if err != nil {
		return fmt.Errorf("detail: %w", err)
	}

	c := detail.Canonical

	fmt.Println(headerStyle.Render(c.Headline))
	fmt.Println()

	fmt.Println(renderKeyValue("Canonical ID", c.CanonicalID))
	fmt.Println(renderKeyValue("Source", c.Source))
	fmt.Println(renderKeyValue("Kind", c.Kind))
	if c.Address != "" {
		fmt.Println(renderKeyValue("Address", c.Address))
	}
	if c.City != "" {
		fmt.Println(renderKeyValue("City", c.City))
	}
	if c.Postal != "" {
		fmt.Println(renderKeyValue("Postal", c.Postal))
	}
	fmt.Println(renderKeyValue("Price", formatPrice(c.Price)))
	fmt.Println(renderKeyValue("Area", formatArea(c.Area)))
	if c.RoomLayout != "" {
		fmt.Println(renderKeyValue("Room Layout", c.RoomLayout))
	}
	if c.URL != "" {
		fmt.Println(renderKeyValue("URL", c.URL))
	}
	if webLink := buildWebLink(webBaseURL, c.CanonicalID); webLink != "" {
		fmt.Println(renderKeyValue("Web", webLink))
	}
	if !c.LastSeenAt.IsZero() {
		fmt.Println(renderKeyValue("Last Seen", c.LastSeenAt.Format("2006-01-02 15:04")))
	}

	if len(detail.CanonicalExtra) > 0 {
		fmt.Println()
		fmt.Println(headerStyle.Render("Details"))
		for _, f := range detail.CanonicalExtra {
			fmt.Println(renderKeyValue(f.Label, f.Value))
		}
	}

	if len(detail.SourceSpecific) > 0 {
		fmt.Println()
		fmt.Println(headerStyle.Render("Source Specific"))
		for _, f := range detail.SourceSpecific {
			fmt.Println(renderKeyValue(f.Label, f.Value))
		}
	}

	if len(detail.Related) > 0 {
		fmt.Println()
		fmt.Println(headerStyle.Render("Related"))
		for _, f := range detail.Related {
			fmt.Println(renderKeyValue(f.Label, f.Value))
		}
	}

	return nil
}

func buildWebLink(base, canonicalID string) string {
	base = strings.TrimSpace(base)
	id := strings.TrimSpace(canonicalID)
	if base == "" || id == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/detail/" + id
}
