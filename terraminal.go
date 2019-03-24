package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var iPipe io.WriteCloser

func main() {
	if len(os.Args) < 2 {
		return
	}

	selfPath, err := filepath.Abs(os.Args[0])
	if err != nil {
		panic(err)
	}

	rootDir := filepath.Dir(selfPath)

	svrArgs := []string{
		"-players", "8",
		"-world", filepath.Join(rootDir, "worlds", os.Args[1]+".wld"),
		"-worldpath", filepath.Join(rootDir, "worlds"),
		"-steam", "-lobby", "friends",
	}
	if len(os.Args) > 2 && os.Args[2] != "_" {
		svrArgs = append(svrArgs, "-port", os.Args[2])
	}
	if len(os.Args) > 3 {
		svrArgs = append(svrArgs, "-pass", os.Args[3])
	}

	svr := exec.Command(
		filepath.Join(rootDir, "TerrariaServer.bin.x86_64"),
		svrArgs...,
	)
	svr.Dir = rootDir

	svr.Stdout = newOutBuffer()
	svr.Stderr = os.Stderr

	iPipe, err = svr.StdinPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		buf := bytes.NewBuffer([]byte{})
		p := make([]byte, 256)
		for {
			sz, err := os.Stdin.Read(p)
			if err != nil {
				return
			}
			handleLine(buf, p[:sz], handleCmd)
		}
	}()

	err = svr.Run()
	if err != nil {
		panic(err)
	}
}

func handleLine(buf *bytes.Buffer, p []byte, cb func(string)) {
	base := 0
	pLen := len(p)
	for i, chr := range p {
		if chr == '\n' {
			buf.Write(p[base:i])
			lineStr := strings.TrimSpace(buf.String())
			if len(lineStr) > 0 {
				cb(lineStr)
			}
			buf.Reset()
			base = i + 1
		}
	}
	if base < pLen {
		buf.Write(p[base:pLen])
	}
}

type outBuffer struct {
	buf *bytes.Buffer
}

func newOutBuffer() *outBuffer {
	return &outBuffer{
		bytes.NewBuffer([]byte{}),
	}
}

func (b *outBuffer) Write(p []byte) (n int, err error) {
	handleLine(b.buf, p, checkOutLine)
	return len(p), nil
}

func (b *outBuffer) Close() error {
	return nil
}

func say(s string) {
	if len(s) == 0 {
		return
	}
	fmt.Fprintln(iPipe, "say", s)
}

func handleCmd(cmd string) {
	if len(cmd) == 0 {
		return
	}
	fmt.Println("[IN]", cmd)
	fmt.Fprintln(iPipe, cmd)
}

var lastOut string

func checkOutLine(s string) {
	s = strings.TrimPrefix(s, ": ")
	if len(s) == 0 {
		return
	}

	fmt.Println("[OUT]", s)

	if s[len(s)-1] == '%' {
		ix := strings.LastIndex(s, ":")
		if ix < 0 {
			ix = strings.LastIndex(s, " ")
		}
		if ix > 0 {
			s = strings.TrimSpace(s[:ix]) + "..."
			if lastOut == s {
				return
			}
			lastOut = s
			say(s)
			return
		}
	}

	if s[0] == '<' {
		nameEnd := strings.Index(s, ">")
		if nameEnd < 0 {
			return
		}
		name := s[1:nameEnd]
		if name == "Server" {
			return
		}
		cont := strings.TrimSpace(s[nameEnd+1:])
		if cont[0] != '/' {
			return
		}
		cmd := cont[1:]
		handleCmd(cmd)
		return
	}

	say(s)
}
