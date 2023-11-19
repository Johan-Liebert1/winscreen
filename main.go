package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3/pkg/media/h264reader"
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
        "-threads", "0",
		// "-preset", "ultrafast", "-tune", "zerolatency", 
        "-f", "mpegts", "udp://127.0.0.1:6969?pkt_size=1316",
	}

	command := exec.Command(cmd, args...)

	fmt.Println(command)

	err := command.Run()

	fmt.Printf("Exited with err: %+v\n", err)
}

func main() {
	go udp()
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

func udp() {
	// Listen for incoming UDP packets
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6969})
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()


    reader, err := h264reader.NewReader(listener)

    if err != nil {
        fmt.Println("Failed to constrcute reader", err)
    }
    
	for {
        nal, err := reader.NextNAL()

        if err != nil {
            fmt.Println("Failed to get get next NAL", err)
            continue
        }

		if ws != nil {
            b := make([]byte, len(nal.Data) + 3)
            b[0] = 0x00
            b[1] = 0x00
            b[2] = 0x00
            b[3] = 0x01

            copy(b[4:], nal.Data)
            err = ws.WriteMessage(websocket.BinaryMessage, b)

			if err != nil {
				continue
			}
		}
	}
}
