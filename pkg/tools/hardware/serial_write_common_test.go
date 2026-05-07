package hardwaretools

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSerialWriteAllRetriesPartialWritesUntilComplete(t *testing.T) {
	now := time.Unix(0, 0)
	calls := 0

	written, err := serialWriteAll(context.Background(), []byte("PING"), time.Second, func() time.Time {
		return now
	}, func(chunk []byte) (int, error) {
		calls++
		now = now.Add(100 * time.Millisecond)
		switch calls {
		case 1:
			if string(chunk) != "PING" {
				t.Fatalf("first chunk = %q, want %q", chunk, "PING")
			}
			return 2, nil
		case 2:
			if string(chunk) != "NG" {
				t.Fatalf("second chunk = %q, want %q", chunk, "NG")
			}
			return 2, nil
		default:
			t.Fatalf("unexpected extra write call %d", calls)
			return 0, nil
		}
	})
	if err != nil {
		t.Fatalf("serialWriteAll() error = %v", err)
	}
	if written != 4 {
		t.Fatalf("serialWriteAll() wrote %d bytes, want 4", written)
	}
}

func TestSerialWriteAllTimesOutAfterZeroByteWrites(t *testing.T) {
	now := time.Unix(0, 0)
	calls := 0

	written, err := serialWriteAll(context.Background(), []byte("A"), 250*time.Millisecond, func() time.Time {
		return now
	}, func(chunk []byte) (int, error) {
		calls++
		now = now.Add(100 * time.Millisecond)
		return 0, nil
	})
	if err == nil || err.Error() != "timeout while writing serial data" {
		t.Fatalf("serialWriteAll() error = %v, want timeout", err)
	}
	if written != 0 {
		t.Fatalf("serialWriteAll() wrote %d bytes, want 0", written)
	}
	if calls != 3 {
		t.Fatalf("write calls = %d, want 3", calls)
	}
}

func TestSerialWriteAllReturnsContextCancellationAfterRetryBoundary(t *testing.T) {
	now := time.Unix(0, 0)
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	written, err := serialWriteAll(ctx, []byte("A"), time.Second, func() time.Time {
		return now
	}, func(chunk []byte) (int, error) {
		calls++
		now = now.Add(100 * time.Millisecond)
		cancel()
		return 0, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("serialWriteAll() error = %v, want context canceled", err)
	}
	if written != 0 {
		t.Fatalf("serialWriteAll() wrote %d bytes, want 0", written)
	}
	if calls != 1 {
		t.Fatalf("write calls = %d, want 1", calls)
	}
}
