package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/zishang520/socket.io/v2/socket"
)

type AttrJson map[string]interface{}
type Connections map[string]*socket.Socket

type Client struct {
	ClientID    string `json:"client"`
	ClientToken string `json:"token"`
}

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

	server := socket.NewServer(nil, nil)
	http.Handle("/socket.io/", server.ServeHandler(nil))
	go http.ListenAndServe(":"+port, nil)
	UserConns = make(Connections)
	AgentConns = make(Connections)
	ListAgents = make(map[string]string)

	//user connect
	server.Of("/user", nil).On("connection", func(clients ...any) {
		s := clients[0].(*socket.Socket)
		s.On("msg", func(datas ...any) {
			msg := Msg{}
			jsonStr, _ := json.Marshal(datas[0])
			json.Unmarshal(jsonStr, &msg)
			fmt.Println(msg)
			if AgentConns[msg.To] != nil {
				AgentConns[msg.To].Emit("msg", msg)
			}
		})
		s.On("disconnect", func(...any) {
		})
		s.On("login", func(datas ...any) {
			client := Client{}
			jsonStr, _ := json.Marshal(datas[0])
			json.Unmarshal(jsonStr, &client)
			fmt.Println(client)

			if (client.ClientID == "") || !(checktoken(client.ClientToken)) {
				log.Println("miss token or client")

			}
			UserConns[client.ClientID] = s
			s.Emit("list_agents", ListAgents)
			log.Println("User connected:", client)

		})

	})

	//agent connect
	server.Of("/agent", nil).On("connection", func(clients ...any) {
		s := clients[0].(*socket.Socket)
		s.On("msg", func(datas ...any) {
			msg := Msg{}
			jsonStr, _ := json.Marshal(datas[0])
			json.Unmarshal(jsonStr, &msg)
			fmt.Println(msg)
			if UserConns[msg.To] != nil {
				UserConns[msg.To].Emit("msg", msg)
			}
		})
		s.On("disconnect", func(...any) {
		})
		s.On("login", func(datas ...any) {
			client := Client{}
			jsonStr, _ := json.Marshal(datas[0])
			json.Unmarshal(jsonStr, &client)
			fmt.Println(client)
			AgentConns[client.ClientID] = s
			ListAgents[client.ClientID] = time.Now().Format("2006.01.02 15:04:05")
			for _, user_conn := range UserConns {
				user_conn.Emit("list_agents", ListAgents)
			}
			log.Println("Agent connected:", client)
		})

	})

	exit := make(chan struct{})
	SignalC := make(chan os.Signal)

	signal.Notify(SignalC, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range SignalC {
			switch s {
			case os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				close(exit)
				return
			}
		}
	}()

	<-exit
	server.Close(nil)
	os.Exit(0)

}
