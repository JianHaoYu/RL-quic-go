package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"
)

const testdata = "test_25000.txt"
const addr = "10.0.0.1:6969"

func main() {
	ip := flag.String("ip", addr, "IP:Port Address")
	flag.Parse()
	server, err := net.Listen("tcp", *ip)
	if err != nil {
		fmt.Printf("Listen() failed, err: %v \n", err)
	}
	conn, err := server.Accept()
	if err != nil {
		fmt.Printf("Accept() failed, err: %v \n", err)
	}
	defer conn.Close()
	fd := openfile(testdata)
	time.Sleep(1 * time.Second)
	PacketNumber := 0
	for {
		packet := fd[:1400]
		_, err := conn.Write(packet)
		if err != nil {
			panic(err)
		}
		fmt.Printf("[Server] SendPacketNumber: %d ,Length: %d ,AtTime: %f  \n", PacketNumber, len(packet), float64(time.Now().UnixNano()/1e3)/1e6)
		PacketNumber = PacketNumber + 1
		fd = fd[1400:]
		if len(fd) == 0 {
			break
		}
	}
	time.Sleep(1 * time.Second)
	fmt.Printf("SendOver\n")
}

func openfile(filename string) []byte {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Println("read file fail", err)
	}
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	f.Close()
	return bytes
}
