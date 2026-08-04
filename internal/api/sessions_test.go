package api

import "testing"

func TestSessionFilter_RejectsInjection(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"source", false},
		{"destination", false},
		{"application", false},
		{"source/><evil", true},
		{"hello world", true},
		{"UPPER_CASE!", true},
		{"", true},
	}
	for _, tc := range cases {
		_, err := buildSessionFilterCmd(tc.in)
		gotErr := err != nil
		if gotErr != tc.wantErr {
			t.Errorf("buildSessionFilterCmd(%q): err=%v, wantErr=%v", tc.in, err, tc.wantErr)
		}
	}
}
