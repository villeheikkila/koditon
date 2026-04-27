package sitemap

import (
	"encoding/xml"
	"strings"
)

const sitemapNamespace = "http://www.sitemaps.org/schemas/sitemap/0.9"

func ExtractLocs(rawXML string) []string {
	decoder := xml.NewDecoder(strings.NewReader(rawXML))
	var locs []string
	var stack []xml.Name
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name)
			if t.Name.Local != "loc" || !isSitemapName(t.Name) || !isSitemapLoc(stack) {
				continue
			}
			var loc string
			if err := decoder.DecodeElement(&loc, &t); err != nil {
				break
			}
			stack = stack[:len(stack)-1]
			loc = strings.TrimSpace(loc)
			if loc != "" {
				locs = append(locs, loc)
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return locs
}

func isSitemapName(name xml.Name) bool {
	return name.Space == "" || name.Space == sitemapNamespace
}

func isSitemapLoc(stack []xml.Name) bool {
	if len(stack) < 2 {
		return false
	}
	parent := stack[len(stack)-2].Local
	if parent != "url" && parent != "sitemap" {
		return false
	}
	for i := len(stack) - 3; i >= 0; i-- {
		switch stack[i].Local {
		case "urlset", "sitemapindex":
			return true
		case "url", "sitemap":
			return false
		}
	}
	return false
}
