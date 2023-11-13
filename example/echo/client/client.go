package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"time"

	"github.com/lucas-clemente/quic-go"
)

// const addr = "127.0.0.1:6868"

const addr = "10.0.0.1:6869"

func main() {
	time.Sleep(10 * time.Millisecond)
	ip := flag.String("ip", addr, "IP:Port Address")
	flag.Parse()
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-echo-example"},
	}
	session, err := quic.DialAddr(*ip, tlsConf, nil)
	if err != nil {
		fmt.Println(err)
		fmt.Println("session start error ...")
	}
	stream, err := session.AcceptStream(context.Background())
	if err != nil {
		panic(err)
	}
	receivedata(stream, *ip)
	time.Sleep(5 * time.Second)
}

func receivedata(stream quic.Stream, ip string) {
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
			nowPacketID := len(echoBuffer) / 1400
			if nowPacketID != lastPacketID {
				fmt.Printf("[Client] GetPacketNumber: %d ,AtTime: %f  \n", nowPacketID, float64(time.Now().UnixNano()/1e3)/1e6)
			}
			lastPacketID = nowPacketID
		}
	}
	fmt.Println("receiveover")
	time.Sleep(5 * time.Second)
	// writefile(echoBuffer, ip)
}

// func writefile(filebytes []byte, ip string) {
// 	err2 := ioutil.WriteFile("./output2.txt"+ip, filebytes, 0666) //写入文件(字节数组)
// 	fmt.Println("receiveover")
// 	if err2 != nil {
// 		panic(err2)
// 	}
// }
