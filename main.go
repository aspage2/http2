package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"http2/session"
	"io"
	"net"
)

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func Handle(
	headers map[string][]string,
	data []byte,
) (map[string][]string, []byte) {
	fmt.Println(headers)
	m := make(map[string][]string)
	m["Content-Type"] = []string{"text/html"}
	return m, []byte("<h1>Hello, world</h1>")
}

func TLSListener(bindAddr string) net.Listener {
	cert := Must(tls.LoadX509KeyPair("certs/cert.pem", "certs/key.pem"))
	var cfg tls.Config
	cfg.Certificates = append(cfg.Certificates, cert)
	cfg.NextProtos = append(cfg.NextProtos, "h2")
	return Must(tls.Listen("tcp", bindAddr, &cfg))
}

func serverMain(bindAddr string, tls bool) {
	var listener net.Listener
	if tls {
		listener = TLSListener(bindAddr)
		fmt.Printf("server available at https://%s\n", bindAddr)
	} else {
		panic("non-tls not implemented")
	}

	for {
		conn := Must(listener.Accept())
		srv := session.NewSession(conn, conn)
		srv.Handle = Handle
		srv.Serve()
	}
}

func NestedBuf(rd io.Reader) {
	buf := bufio.NewReader(rd)
	fmt.Printf("NestedBuf got this char: %c\n", Must(buf.ReadByte()))
}

func main() {
	useTLS := flag.Bool("tls", true, "whether or not to use tls")
	bind := flag.String("bind", ":8000", "host:port authority to listen on")
	flag.Parse()

	serverMain(*bind, *useTLS)
}
