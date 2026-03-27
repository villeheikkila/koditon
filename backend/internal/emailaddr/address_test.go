package emailaddr

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "normalizes casing and whitespace", raw: "  User.Name@Example.COM  ", want: "user.name@example.com"},
		{name: "rejects empty", raw: "   ", wantErr: true},
		{name: "rejects missing at", raw: "user.example.com", wantErr: true},
		{name: "rejects multiple at", raw: "a@@example.com", wantErr: true},
		{name: "rejects whitespace inside", raw: "user @example.com", wantErr: true},
		{name: "rejects control characters", raw: "user@\nexample.com", wantErr: true},
		{name: "rejects display name", raw: "User <user@example.com>", wantErr: true},
		{name: "rejects missing dot in domain", raw: "user@example", wantErr: true},
		{name: "rejects domain with double dot", raw: "user@example..com", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got success: %q", got.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got.String() != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got.String())
			}
		})
	}
}
