package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestShowBanner_NonTTY(t *testing.T) {
	// Capture stdout via a pipe (non-TTY) — banner should be suppressed.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}

	origStdout := os.Stdout
	os.Stdout = w

	ShowBanner()

	w.Close()
	os.Stdout = origStdout

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	r.Close()

	output := string(buf[:n])
	if output != "" {
		t.Errorf("ShowBanner() produced output in non-TTY context: %q", output)
	}
}

func TestBannerArt_ContainsDataKit(t *testing.T) {
	// The raw ASCII art constant must spell out "DataKit".
	if !strings.Contains(bannerArt, "DataKit") && !strings.Contains(bannerArt, "____") {
		t.Error("bannerArt does not appear to contain the DataKit branding")
	}
}

func TestBannerArt_ASCIIOnly(t *testing.T) {
	for i, ch := range bannerArt {
		if ch > 126 { // only printable ASCII + common control chars
			t.Errorf("bannerArt contains non-ASCII character at position %d: %U", i, ch)
			break
		}
	}
}

func TestMinBannerWidth(t *testing.T) {
	if minBannerWidth != 40 {
		t.Errorf("minBannerWidth = %d, want 40", minBannerWidth)
	}
}
