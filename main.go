package main

import (
	"log"
	"net/http"
	"os"

	socketio "github.com/googollee/go-socket.io"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/cors"
)

type AttrJson map[string]interface{}
type Connections map[string]*socketio.Conn

type Msg struct {
	From        string   `json:"from"`
	To          string   `json:"to"`
	CommandType string   `json:"command_type"`
	CommandInfo AttrJson `json:"command_info"`
}

var UserConns Connections
var AgentConns Connections

func main() {
	server := socketio.NewServer(nil)
	UserConns = make(Connections)
	AgentConns = make(Connections)

	//user connect
	server.OnConnect("/user", func(s socketio.Conn) error {
		token := s.RemoteHeader().Get("token")
		client := s.RemoteHeader().Get("client")
		if (token == "") || (client == "") {
			s.Close()
			return nil
		}
		UserConns[client] = &s

		//s.SetContext(client)
		log.Println("User connected:", s.ID(), token, client)

		return nil
	})

	//agent connect
	server.OnConnect("/agent", func(s socketio.Conn) error {
		token := s.RemoteHeader().Get("token")
		client := s.RemoteHeader().Get("client")
		if (token == "") || (client == "") {
			s.Close()
			return nil
		}
		AgentConns[client] = &s
		//s.SetContext(client)
		log.Println("Agent connected:", s.ID(), token, client)
		return nil
	})

	//user send message
	server.OnEvent("/user", "msg", func(s socketio.Conn, msg Msg) {
		log.Println("msg from user:", msg)
		var AgentConn socketio.Conn
		if AgentConns[msg.To] != nil {
			msg.From = s.RemoteHeader().Get("client")
			AgentConn = *AgentConns[msg.To]
			AgentConn.Emit("msg", msg)
			log.Println("send to", msg.To)
		}

	})

	//agent send message
	server.OnEvent("/agent", "msg", func(s socketio.Conn, msg Msg) {
		log.Println("msg from agent:", msg)
		var UserConn socketio.Conn
		if UserConns[msg.To] != nil {
			msg.From = s.RemoteHeader().Get("client")
			UserConn = *UserConns[msg.To]
			UserConn.Emit("msg", msg)
			log.Println("send to", msg.To)
		}

	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("closed:", s.Context(), reason)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	port := os.Getenv("PORT")
	log.Println("start", port)
	handler := cors.Default().Handler(server)
	log.Fatal(http.ListenAndServe(`:`+port, handler))

}
