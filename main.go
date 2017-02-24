package main

import (
	"fmt"
	"time"
	"log"
	"net/http"
	"regexp"
	"io"
	"strings"

	"golang.org/x/net/http2"
)

type User struct {
	nick   string
	roomID string
	lastSent time.Time
}

type Message struct {
	roomID    string
	nick      string
	text      string
	timestamp time.Time
}

type Chatroom struct {
	ID       string
	title    string
	members  []*User
	msg      []*Message
	numOfMsg int
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

func (c Chatroom)addMessage(m Message) {
	if c.msg == nil {
		c.msg = make([]*Message, 100)
	}
	c.msg = append(c.msg, &m)
}

const mainJS = `console.log("hello world");`

const indexHTML = `<html>
<head>
	<title>Hello</title>
		<script src="/main.js"></script>
		</head>
		<body>
		</body>
		</html>
`

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

var rString = regexp.MustCompile(`.+`)

func main() {
	sampleChat := Chatroom{ID: "asdf", title: "fda"}
	chatroomMap["asdf"] = sampleChat

	var srv http.Server
	srv.Addr = "localhost:7072"

	http.HandleFunc("/main.js", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, mainJS)
	})
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {

		pusher, ok := w.(http.Pusher)
		if ok { // Push is supported. Try pushing rather than waiting for the browser.
			if err := pusher.Push("/main.js", nil); err != nil {
				log.Printf("Failed to push: %v", err)
			}
		}
		fmt.Fprintf(w, indexHTML)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, loginHTML)
	})

	http.HandleFunc("/chat/say", func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("roomID")
		nickname := r.URL.Query().Get("nickname")
		text := r.URL.Query().Get("text")

		chatroom, ok := chatroomMap[roomID]

		if !ok {
			log.Printf("roomID doesn't exist")
			return
		}

		message := Message{roomID,nickname,text,time.Now()}

		chatroom.addMessage(message)

	})

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		header := r.Proto
		roomID := r.URL.Query().Get("roomID")
		nickname := r.URL.Query().Get("nickname")
		log.Println(header)
		log.Println(roomID)
		log.Println(nickname)

		clientGone := w.(http.CloseNotifier).CloseNotify()
		w.Header().Set("Content-Type", "text/plain")
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		fmt.Fprintf(w, "# ~1KB of junk to force browsers to start rendering immediately: \n")
		io.WriteString(w, strings.Repeat("# xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n", 13))
		for {
			fmt.Fprintf(w, "%v\n", time.Now())
			w.(http.Flusher).Flush()
			select {
			case <-ticker.C:
				log.Println("tick")
			case <-clientGone:
				log.Println("Client %v disconnected from the clock", r.RemoteAddr)
				return
			}
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
