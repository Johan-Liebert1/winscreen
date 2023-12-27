package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	// "github.com/bluenviron/gortsplib/v4/pkg/format/rtph264"
	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	// "github.com/pion/webrtc/v3/pkg/media/h264reader"
)

var ws *websocket.Conn

func main() {
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func StartServer() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			fmt.Println("Failed to upgrade conn", err)
			os.Exit(1)
		}

		ws = conn
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
