package prompt

import (
	"bytes"
	"testing"
)

func TestIntR(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    int
		wantErr bool
	}{
		{"simple", "42\n", 42, false},
		{"negative", "-3\n", -3, false},
		{"retry-after-invalid", "x\n7\n", 7, false}, // first token invalid, second ok
	}

	for _, tc := range tests {
		tc := tc // capture
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := intR(bytes.NewBufferString(tc.in), "p?")
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestUintR(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    uint
		wantErr bool
	}{
		{"simple", "9\n", 9, false},
		{"zero", "0\n", 0, false},
		{"retry-after-negative", "-1\n8\n", 8, false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := uintR(bytes.NewBufferString(tc.in), "p?")
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestStringR(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"plain", "hello\n", "hello", false},
		{"trim-space", "  hi there   \n", "hi there", false},
		{"empty", "\n", "", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := stringR(bytes.NewBufferString(tc.in), "p?")
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestYesNoR(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    bool
		wantErr bool
	}{
		{"yes-lower", "y\n", true, false},
		{"yes-upper", "YES\n", true, false},
		{"no-mixed", "No\n", false, false},
		{"retry-after-junk", "maybe\nn\n", false, false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := yesNoR(bytes.NewBufferString(tc.in), "continue?")
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
