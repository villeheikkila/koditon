package oauthapi

import (
	"fmt"
	"html/template"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

func renderQRCodeSVG(content string) (template.HTML, error) {
	code, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("create qr code: %w", err)
	}
	code.DisableBorder = true
	bitmap := code.Bitmap()
	size := len(bitmap)
	if size == 0 {
		return "", fmt.Errorf("empty qr bitmap")
	}

	var builder strings.Builder
	builder.Grow(size * size * 8)
	builder.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 `)
	_, _ = fmt.Fprintf(&builder, "%d %d", size, size)
	builder.WriteString(`" shape-rendering="crispEdges" aria-hidden="true">`)
	builder.WriteString(`<rect width="100%" height="100%" fill="#fff"/>`)
	for y, row := range bitmap {
		runStart := -1
		for x, dark := range row {
			if dark {
				if runStart == -1 {
					runStart = x
				}
				continue
			}
			if runStart != -1 {
				_, _ = fmt.Fprintf(&builder, `<rect x="%d" y="%d" width="%d" height="1" fill="#111"/>`, runStart, y, x-runStart)
				runStart = -1
			}
		}
		if runStart != -1 {
			_, _ = fmt.Fprintf(&builder, `<rect x="%d" y="%d" width="%d" height="1" fill="#111"/>`, runStart, y, len(row)-runStart)
		}
	}
	builder.WriteString(`</svg>`)
	return template.HTML(builder.String()), nil
}
