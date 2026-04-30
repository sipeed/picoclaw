//go:build linux || darwin

package hardwaretools

import (
	"context"
	"errors"
	"testing"
	"time"
)

func stubUnixSerialIO(t *testing.T, now *time.Time) {
	t.Helper()

	prevNow := unixSerialNow
	prevOpen := unixSerialOpenPort
	prevClose := unixSerialClosePort
	prevPollRead := unixSerialPollRead
	prevPollWrite := unixSerialPollWrite

	unixSerialNow = func() time.Time {
		return *now
	}
	unixSerialOpenPort = func(cfg serialConfig) (int, error) {
		return 42, nil
	}
	unixSerialClosePort = func(fd int) error {
		return nil
	}
	unixSerialPollRead = prevPollRead
	unixSerialPollWrite = prevPollWrite

	t.Cleanup(func() {
		unixSerialNow = prevNow
		unixSerialOpenPort = prevOpen
		unixSerialClosePort = prevClose
		unixSerialPollRead = prevPollRead
		unixSerialPollWrite = prevPollWrite
	})
}

func TestSerialReadWaitsPastEmptyPollsUntilDeadline(t *testing.T) {
	now := time.Unix(0, 0)
	stubUnixSerialIO(t, &now)

	pollCalls := 0
	unixSerialPollRead = func(fd int, dst []byte, timeout time.Duration) (int, error) {
		pollCalls++
		if timeout > serialPollInterval {
			t.Fatalf("poll timeout = %v, want <= %v", timeout, serialPollInterval)
		}
		now = now.Add(timeout)
		if pollCalls < 4 {
			return 0, nil
		}
		return copy(dst, []byte("OK")), nil
	}

	got, err := serialRead(context.Background(), serialConfig{}, 2, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("serialRead() error = %v", err)
	}
	if string(got) != "OK" {
		t.Fatalf("serialRead() = %q, want %q", got, "OK")
	}
	if pollCalls != 4 {
		t.Fatalf("poll calls = %d, want 4", pollCalls)
	}
}

func TestSerialReadReturnsPromptlyOnContextCancelBetweenPolls(t *testing.T) {
	now := time.Unix(0, 0)
	stubUnixSerialIO(t, &now)

	ctx, cancel := context.WithCancel(context.Background())
	pollCalls := 0
	unixSerialPollRead = func(fd int, dst []byte, timeout time.Duration) (int, error) {
		pollCalls++
		now = now.Add(timeout)
		cancel()
		return 0, nil
	}

	_, err := serialRead(ctx, serialConfig{}, 1, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("serialRead() error = %v, want context canceled", err)
	}
	if pollCalls != 1 {
		t.Fatalf("poll calls = %d, want 1", pollCalls)
	}
}

func TestSerialWriteWaitsPastEmptyPollsUntilReady(t *testing.T) {
	now := time.Unix(0, 0)
	stubUnixSerialIO(t, &now)

	pollCalls := 0
	unixSerialPollWrite = func(fd int, src []byte, timeout time.Duration) (int, error) {
		pollCalls++
		if timeout > serialPollInterval {
			t.Fatalf("poll timeout = %v, want <= %v", timeout, serialPollInterval)
		}
		now = now.Add(timeout)
		switch pollCalls {
		case 1, 2:
			return 0, nil
		default:
			return 1, nil
		}
	}

	written, err := serialWrite(context.Background(), serialConfig{}, []byte("OK"), 500*time.Millisecond)
	if err != nil {
		t.Fatalf("serialWrite() error = %v", err)
	}
	if written != 2 {
		t.Fatalf("serialWrite() wrote %d bytes, want 2", written)
	}
	if pollCalls != 4 {
		t.Fatalf("poll calls = %d, want 4", pollCalls)
	}
}

func TestSerialWriteTimesOutAfterRepeatedEmptyPolls(t *testing.T) {
	now := time.Unix(0, 0)
	stubUnixSerialIO(t, &now)

	unixSerialPollWrite = func(fd int, src []byte, timeout time.Duration) (int, error) {
		now = now.Add(timeout)
		return 0, nil
	}

	written, err := serialWrite(context.Background(), serialConfig{}, []byte("A"), 250*time.Millisecond)
	if err == nil || err.Error() != "timeout while writing serial data" {
		t.Fatalf("serialWrite() error = %v, want timeout", err)
	}
	if written != 0 {
		t.Fatalf("serialWrite() wrote %d bytes, want 0", written)
	}
}
