package main

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestRun_NoArgs_ShowsUsageAndExit1(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := run([]string{}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !bytes.Contains(errOut.Bytes(), []byte("用法")) {
		t.Fatalf("expected usage in stderr, got %q", errOut.String())
	}
}

func TestRun_ScanDispatchesParsedOptions(t *testing.T) {
	orig := executeScan
	t.Cleanup(func() { executeScan = orig })

	called := false
	var got scanOptions
	executeScan = func(opts scanOptions) error {
		called = true
		got = opts
		return nil
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"scan", "-s", "7", "-d", "9s", "-W", "800", "-H", "600", "/tmp/videos"}, &out, &errOut)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", code, errOut.String())
	}
	if !called {
		t.Fatal("expected executeScan to be called")
	}
	if got.root != "/tmp/videos" {
		t.Fatalf("expected root /tmp/videos, got %q", got.root)
	}
	if got.samples != 7 {
		t.Fatalf("expected samples 7, got %d", got.samples)
	}
	if got.minDuration != 9*time.Second {
		t.Fatalf("expected minDuration 9s, got %v", got.minDuration)
	}
	if got.minWidth != 800 || got.minHeight != 600 {
		t.Fatalf("expected resolution 800x600, got %dx%d", got.minWidth, got.minHeight)
	}
}

func TestRun_ScanError_ReturnsCode1(t *testing.T) {
	orig := executeScan
	t.Cleanup(func() { executeScan = orig })

	executeScan = func(opts scanOptions) error {
		return errors.New("boom")
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := run([]string{"scan"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !bytes.Contains(errOut.Bytes(), []byte("boom")) {
		t.Fatalf("expected error output to include boom, got %q", errOut.String())
	}
}
