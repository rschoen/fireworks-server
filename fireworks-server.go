package main

import (
	"bufio"
	"fmt"
	"github.com/rschoen/fireworks-server/lib"
	"net"
	"os"
)

type Server struct {
	games   []*lib.Game
	clients map[net.Conn]*Client
}

type Client struct {
	conn     net.Conn
	ch       chan<- string
	gameid   string
	playerid string
}

type IncomingMessage struct {
	conn net.Conn
	msg  string
}

func main() {
	// check to make sure no other server is running

	// initialize server
	s := Server{}
	s.games = make([]*lib.Game, 0, lib.MaxConcurrentGames)

	// listen for connections
	ln, err := net.Listen("tcp", ":6000")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	msgchan := make(chan IncomingMessage)
	addchan := make(chan Client)
	rmchan := make(chan net.Conn)

	go s.handleMessages(msgchan, addchan, rmchan)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go s.handleConnection(conn, msgchan, addchan, rmchan)
	}
}

func (s *Server) handleConnection(c net.Conn, msgchan chan<- IncomingMessage, addchan chan<- Client, rmchan chan<- net.Conn) {
	ch := make(chan string)

	msgs := make(chan IncomingMessage)

	addchan <- Client{conn: c, ch: ch}

	fmt.Printf("Connection from %v opened.\n", c.RemoteAddr())

	go func() {
		defer close(msgs)

		bufc := bufio.NewReader(c)

		for {
			line, _, err := bufc.ReadLine()
			if err != nil {
				break
			}
			msgs <- IncomingMessage{conn: c, msg: string(line)}
		}
	}()

LOOP:
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				break LOOP
			}
			msgchan <- msg
		case msg := <-ch:
			_, err := c.Write([]byte(msg))
			if err != nil {
				break LOOP
			}
		}
	}

	c.Close()
	fmt.Printf("Connection from %v closed.\n", c.RemoteAddr())
	rmchan <- c
}

func (s *Server) handleMessages(msgchan <-chan IncomingMessage, addchan <-chan Client, rmchan <-chan net.Conn) {
	s.clients = make(map[net.Conn]*Client)

	for {
		select {
		case msg := <-msgchan:
			fmt.Printf("new message: %s\n", msg.msg)
			m, ok := lib.DecodeMove(msg.msg)
			if ok {
				s.sendMoveToGame(m, msg.conn)
			} else {
				fmt.Printf("Malformed message, discarding.")
			}
		case client := <-addchan:
			fmt.Printf("New client: %v\n", client.conn)
			s.clients[client.conn] = new(Client)
			s.clients[client.conn].ch = client.ch
			s.clients[client.conn].conn = client.conn
		case conn := <-rmchan:
			fmt.Printf("Client disconnects: %v\n", conn)
			delete(s.clients, conn)
		}
	}
}

func (s *Server) sendMoveToGame(m lib.Message, conn net.Conn) bool {
	var game *lib.Game
	var player lib.Player

	for _, ongoingGame := range s.games {
		if ongoingGame.GameID == m.Game {
			game = ongoingGame
		}
	}

	if m.MoveType == lib.MoveJoin {
		// create game if it doesn't exist
		if &game == nil {
			game = new(lib.Game)
			game.GameID = m.Game
			game.Initialize()
			s.games = append(s.games, game)
		}

		player := game.GetPlayerByID(m.Player)
		// add player if it doesn't exist
		if &player == nil {
			game.AddPlayer(m.Player)
		}

		// associate player with game, even if it already exists
		// allows rejoining of in-progress games
		s.clients[conn].gameid = m.Game
		s.clients[conn].playerid = m.Player

		return true
	}
	player = game.GetPlayerByID(m.Player)

	if &game == nil {
		fmt.Printf("Attempting to make a move on a nonexistent game.")
		return false
	}

	if &player == nil {
		fmt.Printf("Attempting to make a move with nonexistent player.")
		return false
	}

	if m.MoveType == lib.MoveStart {
		if game.Started {
			fmt.Printf("Attempting to start already started game.")
			return false
		}
		game.Start()
		return true
	}

	if m.MoveType == lib.MovePlay || m.MoveType == lib.MoveDiscard || m.MoveType == lib.MoveHint {
		validMove := game.ProcessMove(m)
		for _, client := range s.clients {
			if client.gameid == game.GameID {
				gameState := game.CreateState(client.playerid)
				client.ch <- lib.EncodeGame(gameState)
			}
		}
		return validMove
	}

	return false
}
