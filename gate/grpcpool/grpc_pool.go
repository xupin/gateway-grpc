package grpcpool

import (
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRpcPool struct {
	gRpcConns []*gRpcConn
	mu        sync.Mutex
	i         int
}

type gRpcConn struct {
	ClientConn *grpc.ClientConn
	mu         sync.Mutex
	streams    int
}

func NewGRpcPool(addr string, size int) (pool *GRpcPool, err error) {
	pool = &GRpcPool{
		gRpcConns: make([]*gRpcConn, size),
	}
	for i := 0; i < size; i++ {
		clientConn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		pool.gRpcConns[i] = &gRpcConn{
			ClientConn: clientConn,
		}
	}
	return
}

func (r *GRpcPool) GetConn() (conn *gRpcConn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	conn = r.gRpcConns[r.i]
	r.i = (r.i + 1) % len(r.gRpcConns)
	return
}

func (r *GRpcPool) GetLeastConn() (conn *gRpcConn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	lastStreams := -1
	for _, gRpcConn := range r.gRpcConns {
		gRpcConn.mu.Lock()
		if lastStreams == -1 || gRpcConn.streams < lastStreams {
			lastStreams = gRpcConn.streams
			conn = gRpcConn
		}
		gRpcConn.mu.Unlock()
	}
	return
}

func (r *gRpcConn) Lock() {
	r.mu.Lock()
}

func (r *gRpcConn) Unlock() {
	r.mu.Unlock()
}

func (r *gRpcConn) IncrStreams() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streams += 1
}

func (r *gRpcConn) DecrStreams() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streams -= 1
}
