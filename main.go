package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"http2/session"
	"net"
	"os"
	"strings"
)

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
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
		fmt.Println("\x1b[31mNEW CONNECTION\x1b[0m")
		ctx := session.NewConnectionContext(conn, conn, session.FuncHandler(Handle))
		ctx.Handler = session.FuncHandler(Handle)
		srv := session.NewDispatcher(ctx)
		go srv.Serve()
	}
}

func Handle(req *session.Request, resp *session.Response) {
	pth := req.GetHeader(":path")
	switch pth {
	case "/":
		Index(resp)
	case "/events":
		Events(resp)
	default:
		resp.SetResponseCode(session.NotFound)
	}
}

func Index(resp *session.Response) {
	s := `
<!DOCTYPE html>
<html>
<head>
    <title>SSE Example</title>
</head>
<body>
    <div id="sse-data"></div>

    <script>
        const eventSource = new EventSource('/events');
        eventSource.onmessage = function(event) {
            const dataElement = document.getElementById('sse-data');
            dataElement.innerHTML += event.data + '<br>';
        };
    </script>
</body>
</html>
	`
	wr := bufio.NewWriter(resp)
	wr.WriteString(s)
	wr.Flush()
}

func Events(resp *session.Response) {
	resp.SetHeader("Access-Control-Allow-Origin", "*")
	resp.SetHeader("Access-Control-Expose-Headers", "Content-Type")

	resp.SetHeader("Content-Type", "text/event-stream")
	resp.SetHeader("Cache-Control", "no-cache")
	resp.SetHeader("Connection", "keep-alive")

	sc := bufio.NewScanner(os.Stdin)

	fmt.Print("> ")
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if t == "q" {
			break
		}
		fmt.Fprintf(resp, "data: %s\n\n", t)
		resp.Flush()
		fmt.Print("> ")
	}
}

func main() {
	useTLS := flag.Bool("tls", true, "whether or not to use tls")
	bind := flag.String("bind", ":8000", "host:port authority to listen on")
	flag.Parse()

	serverMain(*bind, *useTLS)
}
