package main

import (
	"fmt"
	"github.com/rschoen/fireworks-server/lib"
	"net/http"
)

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {

	m, ok := lib.DecodeMove(r.PostFormValue("data"))
	if !ok {
		fmt.Printf("Malformed message, discarding.")
		return
	}
	var game *lib.Game
	for _, ongoingGame := range s.games {
		if ongoingGame.GameID == m.Game {
			game = ongoingGame
		}
	}

	if r.URL.Path[1:] == "join" {
		// create game if it doesn't exist
		if game == nil {
			game = new(lib.Game)
			game.GameID = m.Game
			game.Initialize()
			s.games = append(s.games, game)
		} else {
			fmt.Println(game)
		}

		player := game.GetPlayerByID(m.Player)
		// add player if it doesn't exist
		if player == nil {
			game.AddPlayer(m.Player)
		}
		return
	}

	player := game.GetPlayerByID(m.Player)

	if game == nil {
		fmt.Printf("Attempting to make a move on a nonexistent game.")
		return
	}

	if player == nil {
		fmt.Printf("Attempting to make a move with nonexistent player.")
		return
	}

	if r.URL.Path[1:] == "start" {
		if game.Started {
			fmt.Printf("Attempting to start already started game.")
			return
		}
		game.Start()
		return
	}

	if r.URL.Path[1:] == "move" {
		game.ProcessMove(m)
		return
	}

	fmt.Fprintf(w, "done")
}

type Server struct {
	games []*lib.Game
}

func main() {
	// check to make sure no other server is running

	// initialize server
	s := Server{}
	s.games = make([]*lib.Game, 0, lib.MaxConcurrentGames)

	// listen for connections
	http.HandleFunc("/", s.handler)
	http.ListenAndServe(":8080", nil)

}
