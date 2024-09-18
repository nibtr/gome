package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

const port = "6969"
const maxRead = 512

type MessageType int

const (
	Connected MessageType = iota + 1
	Disconnected
	Chat
)

type Message struct {
	msgType MessageType
	msg     string
	conn    net.Conn
}

type Channel struct {
	internal chan Message // information messages
	out      chan Message // actual chat messages
}

func client(conn net.Conn, channel Channel) {
	defer func() {
		conn.Close()
		channel.internal <- Message{
			msgType: Disconnected,
			msg:     "",
			conn:    conn,
		}
	}()

	channel.internal <- Message{
		msgType: Connected,
		msg:     "",
		conn:    conn,
	}

	buffer := make([]byte, maxRead)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("Error reading from connection: %v", err)
			return
		}
		if n == 0 || strings.TrimSpace(string(buffer[:n])) == "exit" {
			return
		}

		channel.out <- Message{
			msgType: Chat,
			msg:     string(buffer[:n]),
			conn:    conn,
		}
	}
}

func server(channel Channel) {
	// save list of clients
	connectedClients := make(map[string]net.Conn)

	broadcast := func(msg string, exclude string) {
		for k, v := range connectedClients {
			if k != exclude {
				v.Write([]byte(msg))
			}
		}
	}

	for {
		select {
		case data := <-channel.internal:
			ip := data.conn.RemoteAddr().String()
			connectedClients[ip] = data.conn

			switch data.msgType {
			case Connected:
				log.Println("Client: " + ip + " connected")
				broadcast(ip+" has connected\n", ip)
			case Disconnected:
				log.Println("Client: " + ip + " disconnected")
				broadcast(ip+" has disconnected\n", ip)
				delete(connectedClients, ip)
			}
		case data := <-channel.out:
			ip := data.conn.RemoteAddr().String()
			msg := ip + " -- " + data.msg
			broadcast(msg, ip)
		}
	}
}

func main() {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Unable to start server: %v", err)
	}

	log.Printf("Server is listening on port %s", port)

	channel := Channel{
		internal: make(chan Message),
		out:      make(chan Message),
	}
	go server(channel)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Could not accept connection: %v\n", err)
			continue
		}
		go client(conn, channel)
	}
}
