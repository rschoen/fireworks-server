package main

import (
	"flag"
	"fmt"
	"github.com/rschoen/fireworks-server/lib"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	// allow requests to come from anywhere, since clients can be wherever
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// serve client HTTP responses, if it's turned on
	if len(r.URL.Path) < 5 || r.URL.Path[1:5] != "api/" {
		if s.httpServer {
			http.FileServer(http.Dir(s.clientDirectory)).ServeHTTP(w, r)
		}
		return
	}

	m, err := lib.DecodeMove(r.PostFormValue("data"))
	if err != "" {
		log.Println("Received malformed JSON message. Discarding.")
		fmt.Fprintf(w, "Error: Data sent was malformed.")
		return
	}

	var command = r.URL.Path[5:]
	var game *lib.Game
	for _, ongoingGame := range s.games {
		if ongoingGame.GameID == m.Game {
			game = ongoingGame
		}
	}

	if command == "join" {
		// create game if it doesn't exist
		if game == nil {
			game = new(lib.Game)
			game.GameID = m.Game
			var initializationError = game.Initialize()
			if initializationError != "" {
				log.Printf("Failed to initialize game '%s'. Error: %s\n", m.Game, initializationError)
				fmt.Fprintf(w, "Error: Could not initialize game.")
			}
			s.games = append(s.games, game)
			log.Printf("Created new game '%s'\n", m.Game)
		}

		player := game.GetPlayerByID(m.Player)
		// add player if it doesn't exist
		if player == nil {
			game.AddPlayer(m.Player)
			log.Printf("Added player '%s' to game '%s'\n", m.Player, m.Game)
		}
	}

	if command == "status" {
		if game == nil {
			return
		}
	}

	player := game.GetPlayerByID(m.Player)

	if game == nil {
		log.Printf("Attempting to make a move on a nonexistent game '%s'\n", m.Game)
		fmt.Fprintf(w, "Error: The game you're attempting to play no longer exists.")
		return
	}

	if player == nil {
		log.Printf("Attempting to make a move with nonexistent player '%s'\n", m.Player)
		fmt.Fprintf(w, "Error: You're not a member of this game.")
		return
	}

	if command == "start" {
		if game.Started {
			log.Printf("Attempting to start already started game '%s'\n", m.Game)
			fmt.Fprintf(w, "Error: This game has already started.")
			return
		}
		var startError = game.Start()
		if startError != "" {
			log.Printf("Failed to start game '%s'. Error: %s\n", m.Game, startError)
			fmt.Fprintf(w, "Error: Could not start game.")
		}
		log.Printf("Started game '%s'\n", m.Game)
	}

	if command == "move" {
		var processError = game.ProcessMove(m)
		if processError != "" {
			log.Printf("Failed to process move for game '%s'. Error: %s\n", m.Game, processError)
			fmt.Fprintf(w, "Error: Could not process move.")
		}
		log.Printf("Processed move by player '%s' in game '%s'\n", m.Player, m.Game)
	}

	encodedGame, err := lib.EncodeGame(game.CreateState(m.Player))
	if err != "" {
		log.Printf("Failed to encode game '%s'. Error: %s\n", m.Game, err)
		fmt.Fprintf(w, "Error: Could not transmit game state to client.")
	}
	fmt.Fprintf(w, encodedGame)
}

type Server struct {
	games           []*lib.Game
	httpServer      bool
	clientDirectory string
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	// initialize server
	s := Server{}
	s.games = make([]*lib.Game, 0, lib.MaxConcurrentGames)

	// listen for connections
	s.httpServer = *flag.Bool("http-server", true, "Whether to also serve HTTP responses outside API calls.")
	s.clientDirectory = *flag.String("client-directory", lib.ClientDirectory, "Directory to serve HTTP responses from (fireworks-client directory)")
	var port = flag.Int("port", lib.Port, "Port to listen for connections from client.")
	flag.Parse()
	http.HandleFunc("/", s.handler)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))

}
