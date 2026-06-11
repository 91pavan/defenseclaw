// Copyright 2026 Cisco Systems, Inc. and its affiliates
//
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"testing"
)

func TestCheckToolLoop_DetectsConsecutiveRepeat(t *testing.T) {
	r := NewEventRouter(nil, nil, nil, false, nil)

	args := []byte(`{"cmd":"ls -la"}`)

	// First two calls should not trigger.
	if looped, _ := r.checkToolLoop("sess-1", "bash", args); looped {
		t.Fatal("should not detect loop on first call")
	}
	if looped, _ := r.checkToolLoop("sess-1", "bash", args); looped {
		t.Fatal("should not detect loop on second call")
	}

	// Third consecutive call triggers.
	looped, count := r.checkToolLoop("sess-1", "bash", args)
	if !looped {
		t.Fatal("expected loop detection on third consecutive identical call")
	}
	if count != 3 {
		t.Fatalf("expected count=3, got %d", count)
	}
}

func TestCheckToolLoop_ResetsOnDifferentTool(t *testing.T) {
	r := NewEventRouter(nil, nil, nil, false, nil)

	args := []byte(`{"cmd":"ls"}`)

	r.checkToolLoop("sess-1", "bash", args)
	r.checkToolLoop("sess-1", "bash", args)

	// Different tool resets the counter.
	r.checkToolLoop("sess-1", "read_file", args)

	// Now bash again — should not trigger (counter reset).
	if looped, _ := r.checkToolLoop("sess-1", "bash", args); looped {
		t.Fatal("should not detect loop after tool change")
	}
}

func TestCheckToolLoop_ResetsOnDifferentArgs(t *testing.T) {
	r := NewEventRouter(nil, nil, nil, false, nil)

	r.checkToolLoop("sess-1", "bash", []byte(`{"cmd":"ls"}`))
	r.checkToolLoop("sess-1", "bash", []byte(`{"cmd":"ls"}`))

	// Same tool but different args resets.
	r.checkToolLoop("sess-1", "bash", []byte(`{"cmd":"pwd"}`))
	if looped, _ := r.checkToolLoop("sess-1", "bash", []byte(`{"cmd":"pwd"}`)); looped {
		t.Fatal("should not trigger with only 2 identical calls")
	}
}

func TestCheckToolLoop_IsolatesSessions(t *testing.T) {
	r := NewEventRouter(nil, nil, nil, false, nil)

	args := []byte(`{"cmd":"ls"}`)

	r.checkToolLoop("sess-1", "bash", args)
	r.checkToolLoop("sess-1", "bash", args)
	r.checkToolLoop("sess-2", "bash", args) // different session

	// sess-1 third call triggers; sess-2 only has 1.
	if looped, _ := r.checkToolLoop("sess-1", "bash", args); !looped {
		t.Fatal("expected loop for sess-1")
	}
	if looped, _ := r.checkToolLoop("sess-2", "bash", args); looped {
		t.Fatal("should not trigger for sess-2 (only 2 calls)")
	}
}

func TestCheckToolLoop_EmptySessionIgnored(t *testing.T) {
	r := NewEventRouter(nil, nil, nil, false, nil)

	looped, _ := r.checkToolLoop("", "bash", []byte(`{}`))
	if looped {
		t.Fatal("empty session should never trigger loop")
	}
}

func TestIsMemoryTool(t *testing.T) {
	cases := []struct {
		tool   string
		isMem  bool
		isWrit bool
	}{
		{"memory_read", true, false},
		{"memory_write", true, true},
		{"memory_search", true, false},
		{"memory_store", true, true},
		{"memory_save_context", true, true},
		{"memory_create_entry", true, true},
		{"memory_update_note", true, true},
		{"mcp__memory__search", true, false},
		{"read_memory", true, false},
		{"write_memory", true, true},
		{"search_memory", true, false},
		{"store_memory", true, true},
		{"bash", false, false},
		{"read_file", false, false},
	}
	for _, tc := range cases {
		if got := isMemoryTool(tc.tool); got != tc.isMem {
			t.Errorf("isMemoryTool(%q) = %v, want %v", tc.tool, got, tc.isMem)
		}
		if tc.isMem {
			if got := isMemoryWrite(tc.tool); got != tc.isWrit {
				t.Errorf("isMemoryWrite(%q) = %v, want %v", tc.tool, got, tc.isWrit)
			}
		}
	}
}
