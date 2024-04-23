package main

import (
	"crypto/tls"
	"net"
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
		HandleConnection(conn)
	}
}

func main() {
	serverMain()
}
