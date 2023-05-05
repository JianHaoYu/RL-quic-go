package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/lucas-clemente/quic-go"
)

// const addr = "127.0.0.1:6868"

const addr = "10.0.0.1:6868"

func main() {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}
	session, err := quic.DialAddr(addr, tlsConf, nil)
	if err != nil {
		fmt.Println(err)
		fmt.Println("session start error ...")
	}
	stream, err := session.AcceptStream(context.Background())
	if err != nil {
		panic(err)
	}
	receivedata(stream)
}

func receivedata(stream quic.Stream) {
	buf := make([]byte, 1400)
	var echoBuffer []byte
	var end time.Time
	var bytesReceived int
	lastPacketID := -1
	for {
		end = time.Now()
		if err := stream.SetReadDeadline(end.Add(20 * time.Second)); err != nil {
			fmt.Println("Could not set connection read deadline")
		}
		if n, err := stream.Read(buf); err != nil {
			break
		} else {
			bytesReceived += n
			echoBuffer = append(echoBuffer, buf[:n]...)
			nowPacketID := bytesReceived / 1400
			if nowPacketID != lastPacketID {
				fmt.Printf("[Client] GetPacketNumber: %d ,AtTime: %f  \n", nowPacketID, float64(time.Now().UnixNano()/1e3)/1e6)
			}
			lastPacketID = nowPacketID
		}
	}
	writefile(echoBuffer)
}

func writefile(filebytes []byte) {
	err2 := ioutil.WriteFile("./output2.txt", filebytes, 0666) //写入文件(字节数组)
	fmt.Println("receiveover")
	if err2 != nil {
		panic(err2)
	}
}
