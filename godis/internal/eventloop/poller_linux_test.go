package eventloop

import (
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestPollerReadable(t *testing.T) {
	poller, err := NewPollerLinux()
	if err != nil {
		t.Fatal(err)
	}
	defer poller.Close()

	fds, err := unix.Socketpair(
		unix.AF_UNIX,
		unix.SOCK_STREAM,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fds[0])
	defer unix.Close(fds[1])

	// Register fds[0] for Readable events.
	err = poller.Register(fds[0], Readable)
	if err != nil {
		t.Fatal(err)
	}

	// Write data to fds[1].
	_, err = unix.Write(fds[1], []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}

	// Call poller.Wait(100 * time.Millisecond).
	events, err := poller.Wait(100 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that fds[0] is reported as readable.
	found := false

	for _, event := range events {
		if event.FD == fds[0] {
			found = true

			if !event.Readable {
				t.Fatal("fds[0] should be readable")
			}
		}
	}

	if !found {
		t.Fatal("fds[0] was not reported by epoll")
	}
}

func TestModifyEnablesWritableNotifications(t *testing.T) {
	poller, err := NewPollerLinux()
	if err != nil {
		t.Fatal(err)
	}
	defer poller.Close()

	fds, err := unix.Socketpair(
		unix.AF_UNIX,
		unix.SOCK_STREAM,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fds[0])
	defer unix.Close(fds[1])

	err = poller.Register(fds[0], Readable)
	if err != nil {
		t.Fatal(err)
	}

	err = poller.Modify(fds[0], Readable|Writable)
	if err != nil {
		t.Fatal(err)
	}

	events, err := poller.Wait(100 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	found := false

	for _, event := range events {
		if event.FD == fds[0] {
			found = true

			if !event.Writable {
				t.Fatal("fds[0] should be writable")
			}
		}
	}

	if !found {
		t.Fatal("fds[0] was not reported by epoll")
	}
}

func TestUnregisterStopsNotifications(t *testing.T) {
	poller, err := NewPollerLinux()
	if err != nil {
		t.Fatal(err)
	}
	defer poller.Close()

	fds, err := unix.Socketpair(
		unix.AF_UNIX,
		unix.SOCK_STREAM,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fds[0])
	defer unix.Close(fds[1])

	err = poller.Register(fds[0], Readable)
	if err != nil {
		t.Fatal(err)
	}

	err = poller.Unregister(fds[0])
	if err != nil {
		t.Fatal(err)
	}

	_, err = unix.Write(fds[1], []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}

	events, err := poller.Wait(100 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 0 {
		t.Fatalf("expected no events after unregister, got %v", events)
	}
}

func TestWaitReturnsNoEventsWhenTimeoutExpires(t *testing.T) {
	poller, err := NewPollerLinux()
	if err != nil {
		t.Fatal(err)
	}
	defer poller.Close()

	fds, err := unix.Socketpair(
		unix.AF_UNIX,
		unix.SOCK_STREAM,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fds[0])
	defer unix.Close(fds[1])

	err = poller.Register(fds[0], Readable)
	if err != nil {
		t.Fatal(err)
	}

	events, err := poller.Wait(50 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestPollerDetectsHangup(t *testing.T) {
	// 3. Close fds[1].
	// 4. Call Wait().
	// 5. Verify fds[0] has Hangup == true.
	poller, err := NewPollerLinux()
	if err != nil {
		t.Fatal(err)
	}
	defer poller.Close()

	fds, err := unix.Socketpair(
		unix.AF_UNIX,
		unix.SOCK_STREAM,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fds[0])

	err = poller.Register(fds[0], Readable)
	if err != nil {
		t.Fatal(err)
	}

	err = unix.Close(fds[1])
	if err != nil {
		t.Fatal(err)
	}

	events, err := poller.Wait(100 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	found := false

	for _, event := range events {
		if event.FD == fds[0] {
			found = true

			if !event.Hangup {
				t.Fatal("fds[0] should be hangup")
			}
		}
	}

	if !found {
		t.Fatal("fds[0] was not reported by epoll")
	}
}
