package normalize

import "testing"

func TestName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Quận 1", "1"},
		{"Q.1", "1"},
		{"Q. 1", "1"},
		{"q 1", "1"},
		{"HCM", "hcm"},
		{"Hồ Chí Minh", "hồ chí minh"},
		{"Thành phố Hồ Chí Minh", "hồ chí minh"},
		{"TP. Hồ Chí Minh", "hồ chí minh"},
		{"TP.Hồ Chí Minh", "hồ chí minh"},
		{"Tỉnh Bình Dương", "bình dương"},
		{"  Bình   Dương  ", "bình dương"},
		{"Phường Phú Cường", "phú cường"},
		{"P. Phú Cường", "phú cường"},
		{"Huyện Bình Chánh", "bình chánh"},
		{"H. Bình Chánh", "bình chánh"},
		{"Thị xã Dĩ An", "dĩ an"},
		{"TX. Dĩ An", "dĩ an"},
		{"Thị trấn Tân Phú", "tân phú"},
		{"TT. Tân Phú", "tân phú"},
		{"Xã An Xuân", "an xuân"},
		{"X. An Xuân", "an xuân"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := Name(tc.input)
			if got != tc.expected {
				t.Errorf("Name(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}
