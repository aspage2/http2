package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
)

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TLSListener() net.Listener {
	cert := Must(tls.LoadX509KeyPair("certs/cert.pem", "certs/key.pem"))
	var cfg tls.Config
	cfg.Certificates = append(cfg.Certificates, cert)
	cfg.NextProtos = append(cfg.NextProtos, "h2")
	return Must(tls.Listen("tcp", ":8000", &cfg))
}

func PlainListener() net.Listener {
	return Must(net.Listen("tcp", ":8000"))
}

func serverMain() {
	listener := TLSListener()

	for {
		conn := Must(listener.Accept())
		fmt.Println(HandleConnection(conn))
	}
}

func NestedBuf(rd io.Reader) {
	buf := bufio.NewReader(rd)
	fmt.Printf("NestedBuf got this char: %c\n", Must(buf.ReadByte()))
}

func main() {
	data := strings.NewReader("Hello, world")
	NestedBuf(data)

	rest := make([]byte, 1024)
	n, err := data.Read(rest)
	fmt.Printf("%d %e\n", n, err)
}
