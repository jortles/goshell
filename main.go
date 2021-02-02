package main

import (
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
)

var isServer *bool
var isClient *bool

func init() {
	isServer = flag.Bool("server", false, "")
	isClient = flag.Bool("client", false, "")
}

func server() error {
	// Create command
	c := exec.Command("bash")

	// Start the command with a pty.
	ptmx, e := pty.Start(c)
	if e != nil {
		return e
	}
	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	return listen(ptmx)
}

func listen(ptmx *os.File) error {
	fmt.Println("Launching server...")

	// listen on all interfaces
	ln, e := net.Listen("tcp", ":"+os.Args[2])
	if e != nil {
		return e
	}
	// accept connection on port
	conn, e := ln.Accept()
	if e != nil {
		return e
	}

	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize.

	go func() { _, _ = io.Copy(ptmx, conn) }()
	_, e = io.Copy(conn, ptmx)
	return e
}

func client() error {
	// connect to this socket
	conn, e := net.Dial("tcp", os.Args[2])
	if e != nil {
		return e
	}

	oldState, e := terminal.MakeRaw(int(os.Stdin.Fd()))
	if e != nil {
		return e
	}
	defer func() { _ = terminal.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	go func() { _, _ = io.Copy(os.Stdout, conn) }()
	_, e = io.Copy(conn, os.Stdin)
	fmt.Println("Bye!")

	return e
}

func clientAndServer() error {
	flag.Parse()
	if isServer != nil && *isServer {
		fmt.Println("Starting server mode")
		return server()
	}
	if isClient != nil && *isClient {
		fmt.Println("Starting client mode")
		return client()
	} else {
		fmt.Println("Starting client mode")
		return client()
	}
}

func main() {
	if e := clientAndServer(); e != nil {
		fmt.Println(e)
	}
}
