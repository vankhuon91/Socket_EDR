package main

import (
	"log"
	"net/http"
	"os"
	"time"

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

type MsgData struct {
	From string `json:"from"`
	To   string `json:"to"`
	Data string `json:"data"`
}

var UserConns Connections
var AgentConns Connections
var ListAgents map[string]string
var ListUsers map[string]string

func checktoken(token string) bool {
	api := os.Getenv("API") + "/api/v1/token/check"
	req, err := http.NewRequest("GET", api, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	// add a custom header to the request
	// here we specify the header name and value as arguments
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return false
	}
	log.Println("check token:", resp.Status)
	defer resp.Body.Close()
	return resp.Status == "200 OK"
}

func main() {
	port := os.Getenv("PORT")

	server := socketio.NewServer(nil)
	UserConns = make(Connections)
	AgentConns = make(Connections)
	ListAgents = make(map[string]string)

	//user connect
	server.OnConnect("/user", func(s socketio.Conn) error {
		token := s.RemoteHeader().Get("token")
		client := s.RemoteHeader().Get("client")
		if (client == "") || !(checktoken(token)) {
			s.Close()
			return nil
		}
		UserConns[client] = &s

		s.Emit("list_agents", ListAgents)
		//s.SetContext(client)
		log.Println("User connected:", s.ID(), client)

		return nil
	})

	//agent connect
	server.OnConnect("/agent", func(s socketio.Conn) error {
		client := s.RemoteHeader().Get("client")
		// if client == "" {
		// 	s.Close()
		// 	return nil
		// }
		AgentConns[client] = &s
		ListAgents[client] = time.Now().Format("2006.01.02 15:04:05")
		var UserConn socketio.Conn
		for _, user_conn := range UserConns {
			UserConn = *user_conn
			UserConn.Emit("list_agents", ListAgents)
		}
		//s.SetContext(client)
		log.Println("Agent connected:", s.ID(), client)
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
		var client = s.RemoteHeader().Get("client")
		if s.RemoteHeader().Get("token") == "" {
			//agent
			delete(ListAgents, client)
			var UserConn socketio.Conn
			for _, user_conn := range UserConns {
				UserConn = *user_conn
				UserConn.Emit("list_agents", ListAgents)
			}
		} else {
			//user
		}
		log.Println("closed:", s.RemoteHeader().Get("client"), reason)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)

	log.Println("start", port)

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://*", "https://*"},
		AllowCredentials: true,
		// Enable Debugging for testing, consider disabling in production
		Debug: false,
	}).Handler(server)
	log.Fatal(http.ListenAndServe(`:`+port, handler))

}
