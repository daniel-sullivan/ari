package native

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/websocket"
)

// TestConnectCloseConnectedRace connects to a fake ARI server and reads
// Connected while Close runs and the listen goroutine reacts to the closed
// websocket. Under the race detector this fails if the connected flag is
// accessed without synchronisation.
func TestConnectCloseConnectedRace(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/ari/events", websocket.Handler(func(ws *websocket.Conn) {
		var discard []byte
		for websocket.Message.Receive(ws, &discard) == nil {
		}
	}))
	mux.HandleFunc("/ari/asterisk/info", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"system":{"entity_id":"node-a"}}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(&Options{
		Application:  "race-test",
		URL:          srv.URL + "/ari",
		WebsocketURL: "ws://" + strings.TrimPrefix(srv.URL, "http://") + "/ari/events",
		Username:     "u",
		Password:     "p",
	})
	if err := c.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}

	if !c.Connected() {
		t.Fatal("expected a connected client after Connect")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		for i := 0; i < 1000; i++ {
			_ = c.Connected()
		}
	}()
	c.Close()
	<-done

	if c.Connected() {
		t.Fatal("expected a disconnected client after Close")
	}
}
