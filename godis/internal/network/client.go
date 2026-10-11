package network

type Client struct {
	FD          int
	WriteBuffer []byte
	ReadBuffer  []byte
}
