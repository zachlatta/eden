package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os/exec"
	"time"
)

const (
	FromID = "avi@romanoff.me"

	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type msg struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type conn struct {
	ws   *websocket.Conn
	send chan []byte
}

func (c *conn) readPump() {
	defer func() {
		h.unregister <- c
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, msg, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		h.broadcast <- msg
	}
}

func (c *conn) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func (c *conn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

type hub struct {
	conns      map[*conn]bool
	broadcast  chan []byte
	register   chan *conn
	unregister chan *conn
}

var h = hub{
	broadcast:  make(chan []byte),
	register:   make(chan *conn),
	unregister: make(chan *conn),
	conns:      make(map[*conn]bool),
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.conns[c] = true
		case c := <-h.unregister:
			if _, ok := h.conns[c]; ok {
				delete(h.conns, c)
				close(c.send)
			}
		case m := <-h.broadcast:
			for c := range h.conns {
				select {
				case c.send <- m:
				default:
					close(c.send)
					delete(h.conns, c)
				}
			}
		}
	}
}

func renderJSON(w http.ResponseWriter, status int, v interface{}) error {
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(&v)
}

func addDefaultHeaders(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		fn(w, r)
	}
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Msg string `json:"msg"`
		To  string `json:"to"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println(err)
		return
	}

	command := `
tell application "Messages"
	send "%s" to buddy "%s" of service "E:%s"
end tell`

	c := exec.Command("/usr/bin/osascript", "-e",
		fmt.Sprintf(command, req.Msg, req.To, FromID))
	if err := c.Run(); err != nil {
		fmt.Println(err)
		return
	}

	renderJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{"success"})
}

func incomingMsg(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From string `json:"from"`
		Msg  string `json:"msg"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println(err)
		return
	}

	msg, err := json.Marshal(msg{"new_msg", req})
	if err != nil {
		fmt.Println(err)
		return
	}

	h.broadcast <- msg

	renderJSON(w, http.StatusOK, nil)
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	c := &conn{send: make(chan []byte, 256), ws: ws}
	h.register <- c
	go c.writePump()
	c.readPump()
}

func listChats(w http.ResponseWriter, r *http.Request) {
	type chat struct {
		Participants []string `json:"participants"`
		ID           string   `json:"id"`
		FirstMsg     string   `json:"first_message"`
	}

	c := exec.Command("/usr/bin/osascript", "all-chats.applescript")
	byteString, err := c.Output()
	if err != nil {
		fmt.Println(err)
		return
	}

	chats := []chat{}

	if err := json.Unmarshal(byteString, &chats); err != nil {
		fmt.Println(err)
		return
	}

	renderJSON(w, http.StatusOK, chats)
}

func getChat(w http.ResponseWriter, r *http.Request) {
	type chat struct {
		Participants []string `json:"participants"`
		ID           string   `json:"id"`
		FirstMsg     string   `json:"first_message"`
	}

	c := exec.Command("/usr/bin/osascript", "single-chat.applescript")
	byteString, err := c.Output()
	if err != nil {
		fmt.Println(err)
		return
	}

	leChat := chat{}

	if err := json.Unmarshal(byteString, &leChat); err != nil {
		fmt.Println(err)
		return
	}

	renderJSON(w, http.StatusOK, leChat)
}

func createMessage(w http.ResponseWriter, r *http.Request) {
}

func main() {
	go h.run()
	r := mux.NewRouter()
	r.HandleFunc("/send", addDefaultHeaders(sendHandler))
	r.HandleFunc("/incoming_msg", incomingMsg)
	r.HandleFunc("/receive", serveWs)
	r.HandleFunc("/chats", addDefaultHeaders(listChats))
	r.HandleFunc("/chats/{chatID}", getChat)
	r.HandleFunc("/chats/{chatID}/messages", createMessage)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
