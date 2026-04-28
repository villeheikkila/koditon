package properties

import "testing"

func TestFrontdoorImageTemplateVariants(t *testing.T) {
	image := frontdoorImageFromTemplate("//d3ls91xgksobn.cloudfront.net/{imageParameters}/frontdoormedia/images/property/import/1/ORIGINAL.jpeg", "10", "20", "main", "MAIN", ptrInt32(0))
	if image.URL != "https://d3ls91xgksobn.cloudfront.net/1280x854,fit,q80,f=webp/frontdoormedia/images/property/import/1/ORIGINAL.jpeg" {
		t.Fatalf("unexpected large URL: %s", image.URL)
	}
	if image.Variants["gallery"] != "https://d3ls91xgksobn.cloudfront.net/2470x1710,fit,q80,f=webp/frontdoormedia/images/property/import/1/ORIGINAL.jpeg" {
		t.Fatalf("unexpected gallery URL: %s", image.Variants["gallery"])
	}
	if image.Provider != "frontdoor" || image.ProviderID != "20" || image.Role != "MAIN" {
		t.Fatalf("unexpected metadata: %#v", image)
	}
}

func TestShortcutImageVariants(t *testing.T) {
	payload := rawMap{"media": []any{map[string]any{"media_id": float64(123), "ordernr": float64(2), "url_full": "https://cdn/full", "url_large": "https://cdn/large", "url_thumb": "https://cdn/thumb", "tags": []any{"floorplan"}}}}
	media := shortcutMedia(payload)
	if media.MainImage == nil {
		t.Fatal("expected main image")
	}
	if media.MainImage.URL != "https://cdn/large" {
		t.Fatalf("unexpected URL: %s", media.MainImage.URL)
	}
	if media.MainImage.Variants["full"] != "https://cdn/full" || media.MainImage.Variants["thumb"] != "https://cdn/thumb" {
		t.Fatalf("unexpected variants: %#v", media.MainImage.Variants)
	}
	if media.MainImage.Role != "FLOOR_PLAN" {
		t.Fatalf("unexpected role: %s", media.MainImage.Role)
	}
}
