package svid

import "testing"

func TestVendorName(t *testing.T) {
	tests := []struct {
		svid string
		want string
	}{
		// special case: Thunderbolt 3
		{"8087", "Thunderbolt 3"},
		// DisplayPort SVID (standards body, not in svid.ids)
		{"ff01", ""},
		// unknown
		{"0000", ""},
		// some examples from svid.ids
		{"05ac", "Apple, Inc."},
		{"8086", "Intel Corp."},
		{"04e8", "Samsung Electronics Co., Ltd"},
	}
	for _, tt := range tests {
		t.Run(tt.svid, func(t *testing.T) {
			got := VendorName(tt.svid)
			if got != tt.want {
				t.Errorf("VendorName(%q) = %q, want %q", tt.svid, got, tt.want)
			}
		})
	}
}
