package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var subscribers = []chan string{}
var subMux = sync.RWMutex{}

func endpoints() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscribe", subscribeHandler)
	mux.HandleFunc("/broadcast", broadcastHandler)

	return mux
}

func main() {
	log.Fatal(http.ListenAndServe(":8080", endpoints()))
}

func broadcastHandler(w http.ResponseWriter, req *http.Request) {
	msg, _ := ioutil.ReadAll(req.Body)

	subMux.RLock()
	for _, ch := range subscribers {
		ch <- string(msg)
	}
	subMux.RUnlock()
}

func subscribeHandler(w http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	ch := make(chan string, 0)

	subMux.Lock()
	subscribers = append(subscribers, ch)
	subMux.Unlock()

	if err := conn.WriteMessage(websocket.TextMessage, []byte("subscribed")); err != nil {
		log.Fatal(err)
		return
	}

	for {
		msg := <-ch

		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			log.Printf("%v\n", err)
		}
	}
}
