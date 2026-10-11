package network

// Connection state and buffers
// Track connections and manage their read/write buffers.

type Client struct {
	FD          int
	WriteBuffer []byte
	ReadBuffer  []byte
}
