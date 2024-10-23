package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
)

type FileServer struct {
	done  chan os.Signal
	ln    net.Listener
	conns map[string]net.Conn
}

func (fs *FileServer) start() {
	ln, err := net.Listen("tcp", ":3000")
	fs.ln = ln
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Starting the tcp server on:", "PORT", "3000")
	fs.done = make(chan os.Signal, 1)
	signal.Notify(fs.done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			con, err := ln.Accept()
			if err != nil && fs.ln != nil {
				log.Fatal(err)
			}

			go fs.handleConnection(con)

		}
	}()

	sig := <-fs.done
	fmt.Printf("\nOS Signal Recieved: %+v\n", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := fs.Shutdown(ctx); err != nil {
		log.Error("Could not stop the server", "error", err)
	}
}

func (fs *FileServer) Shutdown(ctx context.Context) error {
	fs.ln.Close()
	for con := range fs.conns {
		delete(fs.conns, con)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}

func (fs *FileServer) handleConnection(con net.Conn) {
	fs.conns[con.RemoteAddr().String()] = con

	defer func() {
		con.Close()
		delete(fs.conns, con.RemoteAddr().String())
	}()

	buf := new(bytes.Buffer)
	for {
		var size int64
		binary.Read(con, binary.LittleEndian, &size)
		n, err := io.CopyN(buf, con, size)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(buf.Bytes())
		fmt.Printf("Recieved %d bytes over network \n", n)
	}
}

func sendFile(size int) error {
	file := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, file)
	if err != nil {
		return err
	}

	con, err := net.Dial("tcp", ":3000")
	if err != nil {
		return err
	}

	binary.Write(con, binary.LittleEndian, int64(size))

	n, err := io.CopyN(con, bytes.NewReader(file), int64(size))
	if err != nil {
		return err
	}

	fmt.Printf("Recieved %d bytes!  \n", n)
	return nil
}

func main() {
	go func() {
		time.Sleep(2 * time.Second)
		sendFile(5000)
	}()

	server := &FileServer{
		conns: make(map[string]net.Conn),
	}
	server.start()
}
