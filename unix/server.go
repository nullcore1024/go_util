package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	var file string = "test.sock" //用于 unix domain socket 的文件
start:
	lis, err := net.Listen("unix", file) //开始监听
	if err != nil {                      //如果监听失败，一般是文件已存在，需要删除它
		log.Println("UNIX Domain Socket 创 建失败，正在尝试重新创建 -> ", err)
		err = os.Remove(file)
		if err != nil { //如果删除文件失败 ，要么是权限问题，要么是之前监听不成功，不管是什么 都应该退出程序，不然后面 goto 就死循环了
			log.Fatalln("删除 sock 文件失败！程序退出 -> ", err)
		}
		goto start //删除文件后重新执行一次创建
	} else { //监听成功会直接执行本分支
		fmt.Println("创建 UNIX Domain Socket 成功")
	}

	defer lis.Close() //虽然本次操作不会执行， 不过还是加上比较好

	for {
		conn, err := lis.Accept() //开始接 受数据
		if err != nil {
			log.Println("请求接收错误 -> ", err)
			continue //一个连接错误，不会影响整体的稳定性，忽略就好
		}
		go handle(conn) //开始处理数据
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	for {
		io.Copy(conn, conn) //把发送的数据 转发回去
	}
}
