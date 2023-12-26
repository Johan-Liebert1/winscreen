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
	// playh264()
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

var prevNal = -1

func FindNals(frameBuffer []byte) int {
	three := []byte{0, 0, 1}
	four := []byte{0, 0, 0, 1}

	for i := 0; i < len(frameBuffer)-4; i++ {
		if bytes.Equal(three, frameBuffer[i:i+3]) {
			return i
		} else if bytes.Equal(four, frameBuffer[i:i+4]) {
			return i
		}
	}

	return -1
}

func udp() {
	// Listen for incoming UDP packets
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6969})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Buffer for reading RTP packets
	buffer := make([]byte, 50000)

	frameBuffer := make([]byte, 300_000)

	for {
		// Read a packet
		n, _, err := conn.ReadFrom(buffer)
		if err != nil {
			panic(err)
		}

		// Parse the packet as RTP
		packet := &rtp.Packet{}
		// if err := packet.Unmarshal(buffer[:n]); err != nil {
		// 	panic(err)
		// }

		// fmt.Println(buffer[:n])
		frameBuffer = append(frameBuffer, buffer[:n]...)

		newNalIndex := FindNals(frameBuffer)

		fmt.Println(newNalIndex, len(frameBuffer))

        if newNalIndex == -1  {
            continue
        }

		if prevNal == -1 {
			prevNal = newNalIndex
			continue
		}

		if ws != nil && newNalIndex > prevNal {
			ws.WriteMessage(websocket.BinaryMessage, frameBuffer[prevNal:newNalIndex])
		}

		prevNal = -1
		frameBuffer = []byte{}

		continue

		h264Packet := &codecs.H264Packet{}

		if b, e := h264Packet.Unmarshal(packet.Payload); e == nil && ws != nil && len(b) > 0 {
			// ws.WriteMessage(websocket.BinaryMessage, b)
		} else if e != nil {
			fmt.Println("Error: h264Packet", e)
		} else if ws == nil {
		}
	}
}
