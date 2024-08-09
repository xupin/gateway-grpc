package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/xupin/gateway-grpc/common/network/tcp"
	"github.com/xupin/gateway-grpc/common/proto/rpc/rpcpb"
	"github.com/xupin/gateway-grpc/common/utils/grpcpool"
)

type gateway struct {
	listener net.Listener
	gRpcPool *grpcpool.GRpcPool
}

func main() {
	// 创建rpc连接池
	pool, err := grpcpool.NewGRpcPool("127.0.0.1:8550", 20)
	if err != nil {
		panic(err)
	}
	// 网关服务
	svr := &gateway{
		gRpcPool: pool,
	}
	svr.ListenAndServe(":8080")
}

func (r *gateway) ListenAndServe(addr string) {
	var err error
	if r.listener, err = net.Listen("tcp", addr); err != nil {
		panic(err)
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	// tcp服务
	tcpServer := tcp.Server{
		Addr:    addr,
		Handler: r,
	}
	go tcpServer.Serve(r.listener)
	fmt.Printf("[网关]服务已启动 \n")
	<-ch
	fmt.Printf("[网关]服务已退出 \n")
}

func (r *gateway) ServeTCP(conn net.Conn) {
	// gate conn
	tcpConn := tcp.NewConn()
	if err := tcpConn.Open(conn); err != nil {
		fmt.Printf("[网关]连接失败: %+v \n", err)
		return
	}
	fmt.Printf("[网关]连接成功: %+v \n", tcpConn)
	defer tcpConn.Close()

	// rpc conn
	gRpcConn := r.gRpcPool.GetLeastConn()
	gRpcConn.Lock()
	stream, err := rpcpb.NewGameClient(gRpcConn.ClientConn).Stream(context.Background())
	gRpcConn.Unlock()
	if err != nil {
		fmt.Printf("[网关]rpc连接失败 %+v \n", err)
		return
	}
	fmt.Printf("[网关]rpc连接成功: %+v \n", stream)
	gRpcConn.IncrStreams()
	defer func() {
		stream.CloseSend()
		gRpcConn.DecrStreams()
	}()

	// 处理消息
	go func() {
		for {
			resp, err := stream.Recv()
			if err != nil {
				break
			}
			if err = tcpConn.Write(int(resp.GetType()), resp.GetPayload()); err != nil {
				break
			}
		}
		fmt.Printf("[网关]rpc连接已断开: %+v \n", stream)
	}()
	for {
		msg, err := tcpConn.Receive()
		if err != nil {
			break
		}
		req := &rpcpb.Request{
			Type:    int32(msg.GetType()),
			Payload: msg.GetPayload(),
		}
		if err := stream.Send(req); err != nil {
			break
		}
	}
	fmt.Printf("[网关]连接已断开: %+v \n", tcpConn)
}
