package dataserver

import (
	"context"
	"testing"
	"time"
)

func TestRunGC(t *testing.T) {
	store := NewStore(50 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res, _ := store.Put("short-lived", "", nil, nil)
	RunGC(ctx, store, 10*time.Millisecond, nil)
	time.Sleep(150 * time.Millisecond)

	if _, ok := store.Get(res.ID); ok {
		t.Error("expired resource should have been swept")
	}
}

func TestRunGCRespectsPin(t *testing.T) {
	store := NewStore(50 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res, _ := store.Put("pinned", "", nil, nil)
	store.Pin(res.ID, nil) // indefinite pin
	RunGC(ctx, store, 10*time.Millisecond, nil)
	time.Sleep(150 * time.Millisecond)

	if _, ok := store.Get(res.ID); !ok {
		t.Error("pinned resource should survive GC")
	}
}

func TestRunGCStopsOnCancel(t *testing.T) {
	store := NewStore(1 * time.Hour) // never expires
	ctx, cancel := context.WithCancel(context.Background())
	RunGC(ctx, store, 10*time.Millisecond, nil)
	cancel()
	// No panic = clean exit
}
