package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/websocket"
	// "github.com/pion/webrtc/v3/pkg/media/h264reader"
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
		"-f", "rtp", "rtp://127.0.0.1:6969?pkt_size=1316",
	}

	command := exec.Command(cmd, args...)

	fmt.Println(command)

	err := command.Run()

	fmt.Printf("Exited with err: %+v\n", err)
}

func main() {
	// go ffmpeg()
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

func find0001(thingBytes []byte) int {
    nalPrefix4 := []byte{ 0, 0, 0, 1 }
	nalPrefix3 := []byte{0, 0, 1}

    for i := 0; i < len(thingBytes) - 4; i++ {
        equal4 := bytes.Equal(nalPrefix4, thingBytes[i:i + 4])
        equal3 := bytes.Equal(nalPrefix3, thingBytes[i:i + 3])

        if equal3 || equal4 {
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

	connWrite, err := net.Dial("udp", "127.0.0.1:42069")
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer connWrite.Close()

	// Buffer for reading RTP packets
	buffer := make([]byte, 1500)

	// var frameBuffer []byte = []byte{0x00, 0x00, 0x00, 0x01}

	for {
		// Read a packet
		n, _, err := conn.ReadFrom(buffer)
		if err != nil {
			panic(err)
		}

        index := find0001(buffer[:n])

        if index == -1 {
            fmt.Println("could not find stuff")
            continue
        }

		// Parse the packet as RTP
		// packet := &rtp.Packet{}
		// if err := packet.Unmarshal(buffer[:n]); err != nil {
		// 	panic(err)
		// }

		// frameBuffer = append(frameBuffer, buffer[:n]...)

		//if packet.Marker {
		// fmt.Println(packet.String())

		if ws != nil {
            ws.WriteMessage(websocket.BinaryMessage, buffer[index:n])
		}

		// connWrite.Write(frameBuffer)
		// fmt.Println("write", n, err)

		// frameBuffer = []byte{0x00, 0x00, 0x00, 0x01} // Clear the frame buffer for the next frame
		// }

	}
}
