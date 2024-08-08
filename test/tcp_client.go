package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	// 连接到服务器
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Connected to server")

	// 创建读取用户输入的Reader
	reader := bufio.NewReader(os.Stdin)
	// tcp
	tcpReader := bufio.NewReader(conn)
	for {
		// 从标准输入读取一行
		fmt.Print("Enter message: ")
		message, _ := reader.ReadString('\n')

		// 将消息发送到服务器
		conn.Write([]byte(message))

		// 接收服务器消息
		resp := make([]byte, 1024)
		_, err := tcpReader.Read(resp)
		if err != nil {
			fmt.Println("Error reading response:", err)
			return
		}
		// 打印服务器响应
		fmt.Print("Server response: ", string(resp))
	}
}
