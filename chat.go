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
	clients        map[*net.Conn]bool
	register       chan *net.Conn
	unregister     chan *net.Conn
	broadcast      chan string
}

// Run implements the logic handling connections
func (s *Server) Run() {

	go s.listen()

	for {
		select {
		case conn := <-s.register:
			s.clients[conn] = true
			s.Logger.Println("client connection registered")
			go s.handle(conn)
		case m := <-s.broadcast:
			for conn := range s.clients {
				_, err := fmt.Fprint(*conn, m)
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
			return
		}
		s.register <- &conn
	}
}

func (s *Server) handle(conn *net.Conn) {
	_, err := (*conn).Write([]byte(s.WelcomeMessage + "\n"))
	if err != nil {
		s.Logger.Println(fmt.Sprintf("sending message failed: %v", err))
	}

	r := bufio.NewReader(*conn)

	err = readMessage(r, s.broadcast)
	if err == io.EOF {
		s.Logger.Println("client connection closed")
		s.unregister <- conn
		return
	}
	if err != nil {
		s.Logger.Println(fmt.Sprintf("receiving message failed: %v", err))
	}
}

func readMessage(r *bufio.Reader, broadcast chan string) error {
	for {
		m, err := r.ReadString('\n')
		if err != nil || err == io.EOF {
			return err
		}
		broadcast <- m
	}
}

// Close closes all connection to the server
func (s *Server) Close() {
	s.listener.Close()
}

// ListenAndServe blocks while the server is running
func ListenAndServe(addr string) (err error) {
	s := &Server{
		WelcomeMessage: "Welcome to Chat Room!",
	}

	s.Logger = log.New(os.Stderr, "", log.LstdFlags)
	s.register = make(chan *net.Conn, 1)
	s.unregister = make(chan *net.Conn, 1)
	s.clients = make(map[*net.Conn]bool)
	s.broadcast = make(chan string, 10)

	s.listener, err = net.Listen("tcp", addr)

	if err != nil {
		return err
	}

	s.Logger.Println(fmt.Sprintf("Listening on %v", addr))

	s.Run()

	return nil
}
