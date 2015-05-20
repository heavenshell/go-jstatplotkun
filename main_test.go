package main

import (
	"testing"
)

func TestShouldReadFile(t *testing.T) {
	lines, err := read("./fixtures/jstat_gc.log")
	if err != nil {
		t.Fatalf("fail to read %v", err)
	}
	if len(lines) == 0 {
		t.Fatalf("fail to read")
	}
}
