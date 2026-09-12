package proxy

import "sync"

type ProxyBufferPool struct {
	pool sync.Pool
}

func (p *ProxyBufferPool) Get() []byte {
	return p.pool.Get().([]byte)
}

func (p *ProxyBufferPool) Put(buf []byte) {
	p.pool.Put(buf)
}
