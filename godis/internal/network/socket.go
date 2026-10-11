package network

import "golang.org/x/sys/unix"

type Socket struct{}

// ListenTCP creates a non-blocking TCP listening socket
// and returns its file descriptor.
func ListenTCP(port int) (int, error) {
	fd, err := unix.Socket(
		unix.AF_INET,
		unix.SOCK_STREAM|unix.SOCK_NONBLOCK|unix.SOCK_CLOEXEC,
		0,
	)
	if err != nil {
		return -1, err
	}

	success := false
	defer func() {
		if !success {
			_ = unix.Close(fd)
		}
	}()

	if err := unix.SetsockoptInt(
		fd,
		unix.SOL_SOCKET,
		unix.SO_REUSEADDR,
		1,
	); err != nil {
		return -1, err
	}

	if err := unix.Bind(fd, &unix.SockaddrInet4{
		Port: port,
		Addr: [4]byte{127, 0, 0, 1},
	}); err != nil {
		return -1, err
	}

	if err := unix.Listen(fd, 128); err != nil {
		return -1, err
	}

	success = true
	return fd, nil
}
