package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/websocket"
)

var ws *websocket.Conn

func ffmpeg() {
	cmd := "ffmpeg"

	args := []string{
		"-f", "x11grab",
		"-r", "30",
		"-s", "1920x1080",
		"-i", ":1",
		"-c:v", "h264",
		"-profile:v", "baseline",
		"-pix_fmt", "nv12",
		// "-preset", "ultrafast", "-tune", "zerolatency", 
        "-f", "rtp", "rtp://127.0.0.1:6969",
	}

	command := exec.Command(cmd, args...)

	fmt.Println(command)

	err := command.Run()

	fmt.Printf("Exited with err: %+v\n", err)
}

func main() {
	go ffmpeg()
	go rtp()
	StartServer()
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

func rtp() {
	// Listen for incoming RTP packets
	addr := "127.0.0.1:6969"
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer pc.Close()

	// Connect to WebSocket
	// ws, _, err := websocket.DefaultDialer.Dial("ws://example.com/socket", nil)
	// if err != nil {
	// 	log.Fatal("Dial:", err)
	// }
	// defer ws.Close()

	// Read RTP packets and send them over WebSocket
	for {
		buffer := make([]byte, 1500) // RTP packet size
		_, _, err := pc.ReadFrom(buffer)

		if err != nil {
			log.Fatal(err)
		}

		if ws != nil {
			err = ws.WriteMessage(websocket.BinaryMessage, buffer)

			if err != nil {
				continue
			}
		}
	}
}
