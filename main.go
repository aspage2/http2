package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"http2/hpack"
	"io"
	"net"
	"os"
)

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TLSListener() net.Listener {
	cert := Must(tls.LoadX509KeyPair("certs/server.crt", "certs/server.pem"))
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
		HandleConnection(conn)
	}
}

func main() {
	treeFile := Must(os.Open("huffmantrimmed.txt"))
	tree := Must(hpack.PopulateFromFile(treeFile))
	parser := hpack.NewHeaderParser(tree)

	payloads := []string{
		"828684418cf1e3c2e5f23a6ba0ab90f4ff",
		"828684be58086e6f2d6361636865",
		"828785bf400a637573746f6d2d6b65790c637573746f6d2d76616c7565",
	}

	for i, payload := range payloads {
		fmt.Printf("REQUEST %d\n", i)
		bs := Must(hex.DecodeString(payload))
		p := bufio.NewReader(bytes.NewBuffer(bs))
		for {
			k, v, err := parser.NextHeader(p)
			if err != nil {
				if err == io.EOF {
					break
				}
				panic(err)
			}
			fmt.Printf("%s: %v\n", k, v)
		}
		fmt.Printf("%s\n", parser.DT)
	}
}
