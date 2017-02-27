package main

import (
	"fmt"
	"time"
	"log"
	"net/http"
	"io"
	"strings"

	"golang.org/x/net/http2"
)

type User struct {
	nick     string
	roomID   string
	lastSent time.Time
}

type Event struct {
	nick string
	ch   chan string
}

type Message struct {
	nick      string
	text      string
	timestamp time.Time
}

func (m Message)String() string {
	return fmt.Sprintf("%s : %s , Sent at %s", m.nick, m.text, m.timestamp.String())
}

type Chatroom struct {
	ID       string
	title    string
	members  map[string]chan string
	msg      []*Message
	join     chan Event
	leave    chan string
	say      chan Message
}

func (c Chatroom) run() {
	for {
		select {
		case ev := <-c.join:
			c.members[ev.nick] = ev.ch
			c.Broadcast(fmt.Sprintf("%s has joined\n", ev.nick))
		case nick := <-c.leave:
			delete(c.members, nick)
			c.Broadcast(fmt.Sprintf("%s has been leaved\n", nick))
		case m := <-c.say:
			c.msg = append(c.msg, &m)
			c.Broadcast(fmt.Sprintf("message : %s\n", m.String())) //TODO: string 그대로 보내는 것 개선
		}
	}
}

func (c Chatroom) Broadcast(content string) {
	for key, ch := range c.members {
		log.Println(key)
		ch <- content
	}
}

var (
	userMap = make(map[string]User)
	chatroomMap = make(map[string]Chatroom)
)

func addUser(nick string) User {
	user := User{nick: nick}
	userMap[nick] = user
	return user
}

const loginHTML = `<html>
<head><title>Welcome to CHATTING GO</title>
</head>
<body>
<form action="/chatlist">
Nickname:<br>
<input type="text" name="nickname">
<br>
<input type="submit" value="Submit">
</form>
</body>
</html>`

func main() {
	sampleChat := Chatroom{ID: "asdf", title: "fda", members: make(map[string]chan string),
					msg: make([]*Message, 100), join: make(chan Event), leave: make(chan string), say: make(chan Message)}
	chatroomMap["asdf"] = sampleChat
	go sampleChat.run()

	var srv http.Server
	srv.Addr = "localhost:7072"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, loginHTML)
	})

	http.HandleFunc("/chat/test", func (w http.ResponseWriter, r * http.Request) {
		roomID := r.URL.Query().Get("roomID")
		nickname := r.URL.Query().Get("nickname")
		msg := r.URL.Query().Get("msg")

		chatroom, ok := chatroomMap[roomID]

		if !ok {
			log.Printf("roomID doesn't exist")
			return
		}

		chatroom.say <-Message{nick: nickname, text: msg, timestamp: time.Now()}
	})

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		header := r.Proto
		roomID := r.URL.Query().Get("roomID")
		nickname := r.URL.Query().Get("nickname")
		log.Println(header)
		log.Println(roomID)
		log.Println(nickname)

		chatroom, ok := chatroomMap[roomID]

		log.Println(ok)

		if !ok {
			log.Printf("roomID doesn't exist") //TODO: create chatroom
			return
		}

		clientGone := w.(http.CloseNotifier).CloseNotify()
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "# ~1KB of junk to force browsers to start rendering immediately: \n")
		io.WriteString(w, strings.Repeat("# xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n", 13))

		ch := make(chan string, 100)

		go func(w http.ResponseWriter, r *http.Request, ch chan string) {
			for {
				log.Println("in")
				w.(http.Flusher).Flush()
				select {
				case msg := <-ch:
					fmt.Fprintf(w, msg)
					log.Println("msg is ")
					log.Println(msg)
				case <-clientGone:
					chatroom.leave <- nickname
					log.Println("Client %v disconnected from the clock", r.RemoteAddr)
					return
				}
			}
		}(w, r, ch)

		chatroom.join <- Event{nick: nickname, ch: ch}

		for {
			p := make([]byte, 255)
			log.Println(r.Body.Read(p))
			log.Println(p)
			time.Sleep(100 * time.Second)
		}
	})

	http.HandleFunc("/chatlist/", func(w http.ResponseWriter, r *http.Request) {
		nickname := r.URL.Query().Get("nickname")
		if nickname == "" {
			log.Printf("nickname has no value")
			return
		}

		user, ok := userMap[nickname]
		if ok && user.roomID != "" {
			//TODO: redirect with params
			http.Redirect(w, r, fmt.Sprintf("/chat"), 301)
			return
		}

		if user == (User{}) {
			user = addUser(nickname)
		}

		fmt.Fprintf(w, "ID: ")
		fmt.Fprintf(w, user.nick)
		fmt.Fprintf(w, "\n\nChannel List Below\n")

		for k, _ := range chatroomMap {
			fmt.Fprintf(w, "\n")
			fmt.Fprintf(w, k)
		}

	})

	http2.ConfigureServer(&srv, &http2.Server{})

	// Run crypto/tls/generate_cert.go to generate cert.pem and key.pem.
	// See https://golang.org/src/crypto/tls/generate_cert.go
	log.Fatal(http.ListenAndServeTLS(":7072", "cert.pem", "key.pem", nil))
}
