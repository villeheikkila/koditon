package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"koditon/internal/domain/ads"
)

func RunDetail(ctx context.Context, svc *ads.Service, input, shortcutBase, frontdoorBase, webBaseURL string, out io.Writer) error {
	out = resolveOutput(out)
	canonicalID, err := ads.ResolveInput(input, shortcutBase, frontdoorBase)
	if err != nil {
		return fmt.Errorf("resolve input: %w", err)
	}

	detail, err := svc.DetailByCanonicalID(ctx, canonicalID)
	if err != nil {
		return fmt.Errorf("detail: %w", err)
	}

	c := detail.Canonical

	fmt.Fprintln(out, headerStyle.Render(c.Headline))
	fmt.Fprintln(out)

	fmt.Fprintln(out, renderKeyValue("Canonical ID", c.CanonicalID))
	fmt.Fprintln(out, renderKeyValue("Source", c.Source))
	fmt.Fprintln(out, renderKeyValue("Kind", c.Kind))
	if c.Address != "" {
		fmt.Fprintln(out, renderKeyValue("Address", c.Address))
	}
	if c.City != "" {
		fmt.Fprintln(out, renderKeyValue("City", c.City))
	}
	if c.Postal != "" {
		fmt.Fprintln(out, renderKeyValue("Postal", c.Postal))
	}
	fmt.Fprintln(out, renderKeyValue("Price", formatPrice(c.Price)))
	fmt.Fprintln(out, renderKeyValue("Area", formatArea(c.Area)))
	if c.RoomLayout != "" {
		fmt.Fprintln(out, renderKeyValue("Room Layout", c.RoomLayout))
	}
	if c.URL != "" {
		fmt.Fprintln(out, renderKeyValue("URL", c.URL))
	}
	if webLink := buildWebLink(webBaseURL, c.CanonicalID); webLink != "" {
		fmt.Fprintln(out, renderKeyValue("Web", webLink))
	}
	if !c.LastSeenAt.IsZero() {
		fmt.Fprintln(out, renderKeyValue("Last Seen", c.LastSeenAt.Format("2006-01-02 15:04")))
	}

	if len(detail.CanonicalExtra) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, headerStyle.Render("Details"))
		for _, f := range detail.CanonicalExtra {
			fmt.Fprintln(out, renderKeyValue(f.Label, f.Value))
		}
	}

	if len(detail.SourceSpecific) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, headerStyle.Render("Source Specific"))
		for _, f := range detail.SourceSpecific {
			fmt.Fprintln(out, renderKeyValue(f.Label, f.Value))
		}
	}

	if len(detail.Related) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, headerStyle.Render("Related"))
		for _, f := range detail.Related {
			fmt.Fprintln(out, renderKeyValue(f.Label, f.Value))
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
