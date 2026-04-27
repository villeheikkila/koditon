package sitemap

import (
	"reflect"
	"testing"
)

func TestExtractLocs(t *testing.T) {
	rawXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc> https://example.com/a </loc></url>
	<url><image:loc xmlns:image="http://www.google.com/schemas/sitemap-image/1.1">https://example.com/image.jpg</image:loc></url>
	<url><loc></loc></url>
	<url><loc>https://example.com/b</loc></url>
</urlset>`
	got := ExtractLocs(rawXML)
	want := []string{
		"https://example.com/a",
		"https://example.com/b",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractLocs() = %#v, want %#v", got, want)
	}
}
