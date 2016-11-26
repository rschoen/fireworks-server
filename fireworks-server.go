package main

import (
	"fmt"
	"github.com/rschoen/fireworks-server/lib"
	"math/rand"
	"net/http"
	"time"
)


func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) < 5 || r.URL.Path[1:5] != "api/" {
		http.FileServer(http.Dir(lib.ClientDirectory)).ServeHTTP(w, r)
		return
	}
	
	m, ok := lib.DecodeMove(r.PostFormValue("data"))
	if !ok {
		fmt.Printf("Malformed message, Discarding.")
		fmt.Fprintf(w, "Error: malformed JSON message.")
		return
	}
	var game *lib.Game
	for _, ongoingGame := range s.games {
		if ongoingGame.GameID == m.Game {
			game = ongoingGame
		}
	}

	if r.URL.Path[5:] == "join" {
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
		} else {
			fmt.Println(player)
		}
		
		fmt.Fprintf(w, lib.EncodeGame(game.CreateState(m.Player)))
		return
	}
	
	if r.URL.Path[5:] == "status" {
		if game == nil {
			fmt.Printf("Discarding status check for nonexistent game.")
			return
		}
		fmt.Fprintf(w, lib.EncodeGame(game.CreateState(m.Player)))
		return
	}

	player := game.GetPlayerByID(m.Player)

	if game == nil {
		fmt.Printf("Attempting to make a move on a nonexistent game.")
		fmt.Fprintf(w, "Error: Attempted to make a move on nonexistent game.",)
		return
	}

	if player == nil {
		fmt.Printf("Attempting to make a move with nonexistent player.")
		fmt.Fprintf(w, "Error: Attempting to make a move with nonexistent player.",)
		return
	}

	if r.URL.Path[5:] == "start" {
		if game.Started {
			fmt.Printf("Attempting to start already started game.")
			fmt.Fprintf(w, "Error: Attempting to start already started game.",)
			return
		}
		game.Start()
		fmt.Fprintf(w, lib.EncodeGame(game.CreateState(m.Player)))
		return
	}

	if r.URL.Path[5:] == "move" {
		game.ProcessMove(m)
		fmt.Printf("Global game state: %#v\n\n", *game)
		fmt.Fprintf(w, lib.EncodeGame(game.CreateState(m.Player)))
		return
	}	
	
    fmt.Fprintf(w, "done")
}


type Server struct {
	games   []*lib.Game
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	// initialize server
	s := Server{}
	s.games = make([]*lib.Game, 0, lib.MaxConcurrentGames)

	// listen for connections
	http.HandleFunc("/", s.handler)
    http.ListenAndServe(":"+lib.Port, nil)

}

