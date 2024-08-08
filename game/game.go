package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/xupin/gateway-grpc/common/proto/rpc/rpcpb"
	"google.golang.org/grpc"
)

type server struct {
	rpcpb.UnimplementedGameServer
	listener net.Listener
	exit     chan bool
}

func main() {
	// 游戏服务
	svr := &server{
		exit: make(chan bool, 1),
	}
	// 启动服务
	svr.Start()
}

func (r *server) Start() {
	// 监听
	r.ListenAndServe(":8550")
}

func (r *server) Shutdown() {
	// 关闭监听
	r.listener.Close()
	// 停机前的处理
	done := make(chan bool, 1)
	go func() {
		// 玩家下线
		// playersvr.Ins.Shutdown()
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(60 * time.Second):
	}
	r.exit <- true
}

func (r *server) ListenAndServe(addr string) {
	var err error
	if r.isChild() {
		// fd预留：0stdin、1stdout、2stderr
		f := os.NewFile(3, "")
		r.listener, err = net.FileListener(f)
	} else {
		r.listener, err = net.Listen("tcp", addr)
	}
	if err != nil {
		panic(err)
	}

	// 监听信道
	go r.handleSignals()

	// rpc服务
	grpcServer := grpc.NewServer()
	rpcpb.RegisterGameServer(grpcServer, r)
	go grpcServer.Serve(r.listener)

	// 通知父进程退出
	if r.isChild() {
		syscall.Kill(syscall.Getppid(), syscall.SIGTERM)
	}
	fmt.Printf("[游戏]服务已启动 \n")
	<-r.exit

	// 关闭rpc服务，30秒后强制关闭
	go grpcServer.Stop()
	// <-time.After(1 * time.Second)
	fmt.Printf("[游戏]服务%d已关闭 \n", syscall.Getpid())
}

func (r *server) Stream(stream rpcpb.Game_StreamServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		fmt.Printf("Received message: %s\n", msg.GetPayload())
		resp := &rpcpb.Response{
			Type:    0,
			Payload: []byte("Resp " + string(msg.GetPayload())),
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

// 重载
func (r *server) reload() error {
	listener, ok := r.listener.(*net.TCPListener)
	if !ok {
		return errors.New("listener is not tcp listener")
	}
	f, err := listener.File()
	if err != nil {
		return err
	}
	cmd := exec.Command(os.Args[0])
	cmd.Env = []string{
		"SERVER_CHILD=1",
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// 文件描述符
	cmd.ExtraFiles = []*os.File{f}
	// 执行
	return cmd.Start()
}

// 监听信道
func (r *server) handleSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	for {
		sig := <-ch
		fmt.Printf("[游戏]服务%d接收信号 %+v \n", syscall.Getpid(), sig)
		switch sig {
		case syscall.SIGINT:
			// 外部停机
			r.Shutdown()
		case syscall.SIGTERM:
			// 子进程通知停机
			r.Shutdown()
		case syscall.SIGHUP:
			if err := r.reload(); err != nil {
				panic(err)
			}
		}
	}
}

// 子进程
func (r *server) isChild() bool {
	return os.Getenv("SERVER_CHILD") != ""
}
