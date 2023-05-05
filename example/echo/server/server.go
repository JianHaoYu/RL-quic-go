package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/lucas-clemente/quic-go"
)

// const addr = "127.0.0.1:6868"

const addr = "10.0.0.1:6868"
const testdata = "test_25000.txt"

func main() {
	listener, err := quic.ListenAddr(addr, generateTLSConfig(), nil)
	if err != nil {
		panic(err)
	}
	session, err := listener.Accept(context.Background())
	if err != nil {
		fmt.Println("session open failed")
	}
	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		panic(err)
	}
	time.Sleep(2 * time.Second)
	fd := openfile(testdata)
	PacketNumber := 0
	for {
		packet := fd[:1400]
		_, err := stream.Write(packet)
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
	fmt.Println("sendover")
}

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
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
