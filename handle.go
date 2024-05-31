package main

import (
	"bufio"
	"fmt"
	"http2/session"
	"net"
	"os"
	"strings"
)

func Handler(headers map[string][]string, data []byte) (map[string][]string, []byte) {
	path := strings.TrimLeft(headers[":path"][0], "/")
	data, err := os.ReadFile(path)
	hl := make(map[string][]string)
	if err != nil {
		fmt.Println(err)
		hl[":status"] = []string{"500"}
		return hl, nil
	}
	hl[":status"] = []string{"200"}
	hl["content-type"] = []string{"text/plain"}

	return hl, data
}

func HandleConnection(conn net.Conn) error {
	defer conn.Close()
	// Buffer the incoming connection.
	sess := session.NewSession(bufio.NewReader(conn), conn)
	sess.Handle = Handler
	return sess.Serve()
}
