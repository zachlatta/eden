package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
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
		if r.Method == "OPTIONS" {
			return
		}
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

	var formattedMsg struct {
		UserID string `json:"userId"`
		Text   string `json:"text"`
	}

	formattedMsg.UserID = req.From
	formattedMsg.Text = req.Msg

	msg, err := json.Marshal(msg{"new_msg", formattedMsg})
	if err != nil {
		fmt.Println(err)
		return
	}

	h.broadcast <- msg

	renderJSON(w, http.StatusOK, msg)
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

type chatMsg struct {
	Text   string `json:"text"`
	UserID string `json:"user_id"`
}

type chat struct {
	Participants []string `json:"participants"`
	ID           string   `json:"id"`
	FirstMsg     string   `json:"first_message"`
}

func listChats(w http.ResponseWriter, r *http.Request) {
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
	id := mux.Vars(r)["chatID"]

	c := exec.Command("/usr/bin/osascript", "-e", fmt.Sprintf(getSingleChatAS, id))
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
	chatID := mux.Vars(r)["chatID"]

	var msg struct {
		Text string `json:"text"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		fmt.Println(err)
		return
	}

	command := `
tell application "Messages"
	repeat with aChat in text chats
		if id of aChat is equal to "%s" then
			send "%s" to aChat
		end if
	end repeat
end tell`

	c := exec.Command("/usr/bin/osascript", "-e",
		fmt.Sprintf(command, chatID, msg.Text))
	if err := c.Run(); err != nil {
		fmt.Println(err)
		return
	}

	renderJSON(w, http.StatusOK, struct {
		ID string `json:"id"`
	}{chatID})
}

func main() {
	go h.run()
	r := mux.NewRouter()
	r.HandleFunc("/send", addDefaultHeaders(sendHandler))
	r.HandleFunc("/incoming_msg", incomingMsg)
	r.HandleFunc("/receive", addDefaultHeaders(serveWs))
	r.HandleFunc("/chats", addDefaultHeaders(listChats))
	r.HandleFunc("/chats/{chatID}", addDefaultHeaders(getChat))
	r.HandleFunc("/chats/{chatID}/messages", addDefaultHeaders(createMessage))
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var getSingleChatAS = `
-- JSON Encoding From https://github.com/mgax/applescript-json

on encode(value)
	set type to class of value
	if type = integer or type = boolean then
		return value as text
	else if type = text then
		return encodeString(value)
	else if type = list then
		return encodeList(value)
	else if type = script then
		return value's toJson()
	else
		error "Unknown type " & type
	end if
end encode


on encodeList(value_list)
	set out_list to {}
	repeat with value in value_list
		copy encode(value) to end of out_list
	end repeat
	return "[" & join(out_list, ", ") & "]"
end encodeList


on encodeString(value)
	set rv to ""
	repeat with ch in value
		if id of ch = 34 then
			set quoted_ch to "\\\""
		else if id of ch = 92 then
			set quoted_ch to "\\\\"
		else if id of ch ≥ 32 and id of ch < 127 then
			set quoted_ch to ch
		else
			set quoted_ch to "\\u" & hex4(id of ch)
		end if
		set rv to rv & quoted_ch
	end repeat
	return "\"" & rv & "\""
end encodeString


on join(value_list, delimiter)
	set original_delimiter to AppleScript's text item delimiters
	set AppleScript's text item delimiters to delimiter
	set rv to value_list as text
	set AppleScript's text item delimiters to original_delimiter
	return rv
end join


on hex4(n)
	set digit_list to "0123456789abcdef"
	set rv to ""
	repeat until length of rv = 4
		set digit to (n mod 16)
		set n to (n - digit) / 16 as integer
		set rv to (character (1 + digit) of digit_list) & rv
	end repeat
	return rv
end hex4


on createDictWith(item_pairs)
	set item_list to {}
	
	script Dict
		on setkv(key, value)
			copy {key, value} to end of item_list
		end setkv
		
		on toJson()
			set item_strings to {}
			repeat with kv in item_list
				set key_str to encodeString(item 1 of kv)
				set value_str to encode(item 2 of kv)
				copy key_str & ": " & value_str to end of item_strings
			end repeat
			return "{" & join(item_strings, ", ") & "}"
		end toJson
	end script
	
	repeat with pair in item_pairs
		try
			Dict's setkv(item 1 of pair, item 2 of pair)
		end try
	end repeat
	
	return Dict
end createDictWith


on createDict()
	return createDictWith({})
end createDict

tell application "Messages"
	set textChats to text chats
end tell

set chatsList to []
repeat with aChat in textChats
	
	if id of aChat is equal to "%s" then
		
		set chatObj to createDict()
		
		set userList to []
		tell application "Messages"
			
			set chatSubject to (subject of aChat)
			if chatSubject is missing value then
				set chatSubject to ""
			end if
			
			set chatId to (id of aChat)
			repeat with aParticipant in (get participants of aChat)
				# set userData to {{"first_name", (first name of aParticipant as text)}, {"last_name", (last name of aParticipant as string)}, {"handle", (handle of aParticipant as string)}}
				set firstName to (first name of aParticipant)
				if firstName is missing value then
					set firstName to ""
				end if
				set lastName to (last name of aParticipant)
				if lastName is missing value then
					set lastName to ""
				end if
				set userData to "{\"first_name\":\"" & firstName & "\", \"last_name\":\"" & lastName & "\", \"handle\":\"" & (handle of aParticipant) & "\"}"
				
				set userList to userList & userData
			end repeat
			
		end tell
		
		chatObj's setkv("participants", userList)
		chatObj's setkv("id", chatId)
		chatObj's setkv("first_message", chatSubject)
		set chatsList to chatsList & chatObj
		#log quoted form of encode(chatObj)
		
	end if
	
end repeat

do shell script "echo " & quoted form of encode(chatObj)
`
