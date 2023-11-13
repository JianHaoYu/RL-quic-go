package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"time"
)

const addr = "10.0.0.1:6969"

func main() {
	ip := flag.String("ip", addr, "IP:Port Address")
	flag.Parse()
	conn, err := net.Dial("tcp", *ip)
	if err != nil {
		fmt.Println("Net Dial err : ", err)
		return
	}
	defer conn.Close() // 关闭TCP连接
	buf := make([]byte, 1400)
	var echoBuffer []byte
	var end time.Time
	lastPacketID := -1
	for {
		end = time.Now()
		if err := conn.SetReadDeadline(end.Add(20 * time.Second)); err != nil {
			fmt.Println("Could not set connection read deadline")
		}
		if n, err := conn.Read(buf); err != nil {
			break
		} else {
			echoBuffer = append(echoBuffer, buf[:n]...)
			nowPacketID := len(echoBuffer) / 1400
			if nowPacketID != lastPacketID {
				fmt.Printf("[Client] GetPacketNumber: %d ,AtTime: %f  \n", nowPacketID, float64(time.Now().UnixNano()/1e3)/1e6)
			}
			lastPacketID = nowPacketID
		}
	}
	writefile(echoBuffer, *ip)
}

func writefile(filebytes []byte, ip string) {
	err2 := ioutil.WriteFile("./output2.txt", filebytes, 0666) //写入文件(字节数组)
	fmt.Println("receiveover")
	if err2 != nil {
		panic(err2)
	}
}
