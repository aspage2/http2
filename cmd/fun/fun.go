package main

import (
	"bufio"
	"fmt"
	"http2/pkg/bodystream"
	"io"
	"os"
	"strings"
)

func consumer(bs *bodystream.BodyStream, doneC chan struct{}) {
	defer close(doneC)

	data, err := io.ReadAll(bs)
	if err != nil && err != io.EOF {
		fmt.Println(err)
		return
	}

	fmt.Println("MESSAGE: ", string(data))
}

func main() {
	bs := bodystream.NewBodyStream()
	doneC := make(chan struct{})
	go consumer(bs, doneC)

	buf := bufio.NewWriter(bs)
	sc := bufio.NewScanner(os.Stdin)

	for sc.Scan() {
		line := sc.Text()
		if strings.ToLower(line) == "q" {
			break
		}
		buf.WriteString(line)
		buf.Flush()
	}
	if err := sc.Err(); err != nil {
		fmt.Println("Error: ", err)
		close(doneC)
	}
	bs.Close()

	<-doneC
	fmt.Println("we're done here")
}
