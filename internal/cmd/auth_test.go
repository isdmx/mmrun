package cmd

import "testing"

func TestNeedsMFA(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"", false},
		{"mfa authentication required", true},
		{"Multi-factor authentication is required", true},
		{"invalid password", false},
	}
	for _, c := range cases {
		if got := needsMFA(c.msg); got != c.want {
			t.Errorf("needsMFA(%q) = %v, want %v", c.msg, got, c.want)
		}
	}
}
