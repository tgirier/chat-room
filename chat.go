// Package chat implements a chat server & client
package chat

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/sirupsen/logrus"
)

// Server represents a chat server
type Server struct {
	Logger         logrus.StdLogger // with standard logger can be extended with logrus
	WelcomeMessage string
	listener       net.Listener
	clients        map[*net.Conn]string
	register       chan *Connection
	unregister     chan *net.Conn
	broadcast      chan string
}

// Connection represents a user connection
type Connection struct {
	conn     *net.Conn
	username string
}

// Run implements the logic handling connections
func (s *Server) Run() {

	go s.listen()

	for {
		select {
		case c := <-s.register:
			s.clients[c.conn] = c.username
			s.Logger.Println("client connection registered")
			go s.handle(c)
		case msg := <-s.broadcast:
			for conn := range s.clients {
				_, err := fmt.Fprint(*conn, msg)
				if err != nil {
					s.Logger.Println(fmt.Sprintf("sending message failed: %v", err))
				}
			}
		case conn := <-s.unregister:
			delete(s.clients, conn)
			s.Logger.Println("client connection unregistered")
		}
	}
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.Logger.Println(fmt.Sprintf("connection failed: %v", err))
			continue
		}

		_, err = fmt.Fprintf(conn, s.WelcomeMessage+"\n")
		if err != nil {
			s.Logger.Println(fmt.Sprintf("sending message failed: %v", err))
			continue
		}

		r := bufio.NewReader(conn)

		username, err := r.ReadString('\n')
		if err != nil {
			s.Logger.Println(fmt.Sprintf("reading username failed: %v", err))
			continue
		}

		c := &Connection{
			conn:     &conn,
			username: username,
		}

		s.register <- c
	}
}

func (s *Server) handle(c *Connection) {
	r := bufio.NewReader(*c.conn)

	for {
		msg, err := r.ReadString('\n')
		if err == io.EOF {
			s.Logger.Println("client connection closed")
			s.unregister <- c.conn
			break
		}
		if err != nil {
			s.Logger.Println(fmt.Sprintf("receiving message failed: %v", err))
		}
		s.broadcast <- msg
	}
}

// Close closes all connection to the server
func (s *Server) Close() {
	s.listener.Close()
}

// ListenAndServe blocks while the server is running
func ListenAndServe(addr string) (err error) {
	s := &Server{
		WelcomeMessage: "Welcome to Chat Room! Please enter your username:",
	}

	s.Logger = log.New(os.Stderr, "", log.LstdFlags)
	s.register = make(chan *Connection, 1)
	s.unregister = make(chan *net.Conn, 1)
	s.clients = make(map[*net.Conn]string)
	s.broadcast = make(chan string, 10)

	s.listener, err = net.Listen("tcp", addr)

	if err != nil {
		return err
	}

	s.Logger.Println(fmt.Sprintf("Listening on %v", addr))

	s.Run()

	return nil
}
