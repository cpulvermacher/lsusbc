// Package svid decodes SVID strings to vendor names
package svid

import (
	_ "embed"
	"strings"
)

//go:embed svid.ids
var svidIDs string

var vendors map[string]string

func init() {
	vendors = make(map[string]string)
	for line := range strings.Lines(svidIDs) {
		line = strings.TrimRight(line, "\r")
		if len(line) < 5 || line[0] == '#' {
			continue
		}
		id := strings.ToLower(strings.TrimSpace(line[:4]))
		name := strings.TrimSpace(line[4:])
		vendors[id] = name
	}
}

// VendorName returns a human-readable name for the given SVID (hex string, e.g. "8087").
// Returns empty string if unknown.
func VendorName(svid string) string {
	svid = strings.ToLower(strings.TrimSpace(svid))
	// e.g. https://www.infineon.com/assets/row/public/documents/30/316/infineon-ez-pd-ccgx-host-sdk-user-guide-productqualificationreport-en.pdf
	if svid == "8087" {
		return "Thunderbolt 3"
	}
	return vendors[svid]
}
