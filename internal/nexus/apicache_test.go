package nexus

import (
	"testing"
	"time"
)

func TestAPICacheHit(t *testing.T) {
	c := newAPICache(time.Hour)
	c.set("a", []byte(`{"x":1}`))
	b, ok := c.get("a")
	if !ok {
		t.Fatal("expected hit")
	}
	if string(b) != `{"x":1}` {
		t.Fatalf("got %q", b)
	}
}

func TestAPICacheExpiry(t *testing.T) {
	c := newAPICache(20 * time.Millisecond)
	c.set("a", []byte("1"))
	if _, ok := c.get("a"); !ok {
		t.Fatal("expected hit before expiry")
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := c.get("a"); ok {
		t.Fatal("expected miss after expiry")
	}
}

func TestParseCacheTTL(t *testing.T) {
	tests := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"", 24 * time.Hour, false},
		{"off", 0, false},
		{"0", 0, false},
		{"12h", 12 * time.Hour, false},
		{"bad", 0, true},
	}
	for _, tt := range tests {
		d, err := parseCacheTTL(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseCacheTTL(%q): want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseCacheTTL(%q): %v", tt.in, err)
			continue
		}
		if d != tt.want {
			t.Errorf("parseCacheTTL(%q) = %v, want %v", tt.in, d, tt.want)
		}
	}
}
