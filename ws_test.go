package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSubscribe(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(subscribeHandler))
	defer s.Close()

	subURL := "ws" + strings.TrimPrefix(s.URL, "http")

	wg := &sync.WaitGroup{}

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			ws, _, err := websocket.DefaultDialer.Dial(subURL, nil)
			if err != nil {
				t.Fatalf("%v", err)
			}
			defer ws.Close()

			_, msg, err := ws.ReadMessage()
			if err != nil {
				t.Fatalf("%v", err)
			}
			if string(msg) != "subscribed" {
				t.Error("bad handshake")
			}

			msgType, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}

			if msgType != websocket.TextMessage {
				t.Error("expected type TextMessage")
			}
			if string(msg) != "hello" {
				t.Errorf("expected 'hello' got '%s'", msg)
			}
			fmt.Printf("got: %s\n", msg)

			wg.Done()
		}(wg)
	}

	// there is a race condition here, but it's within the context of the test case.
	// this sleep is necessary to give up the context so that the above goroutines
	// can run and establish the subscriptions.  There's no deadlock race condition
	// in the code under test, but there is a race condition here, in that if the
	// broadcast is sent before the subscribers are listening, our WaitGroup will
	// not release.
	time.Sleep(10 * time.Millisecond)
	req, err := http.NewRequest("POST", "/broadcast", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatalf("%v", err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(broadcastHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("broadcast handler returned status %d", status)
	}

	if timeout := waitTimeout(wg, 1*time.Second); timeout {
		t.Errorf("Timeout waiting for subscribers to ack")
	}
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
