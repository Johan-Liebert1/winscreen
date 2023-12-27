package main

import (
	"errors"
	"io"
	"log"
	"net"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type WebRTCManager struct {
	videoTrack        *webrtc.TrackLocalStaticRTP
	audioTrack        *webrtc.TrackLocalStaticSample
	audioCodec        webrtc.RTPCodecParameters
	api               *webrtc.API
	locked            bool
	started           bool
	videoListener     *net.UDPConn
	resetSequenceChan chan bool
}

func New() *WebRTCManager {
	return &WebRTCManager{
		started:           false,
		resetSequenceChan: make(chan bool),
	}
}

func (manager *WebRTCManager) Start() {
	var err error

	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5004})
	if err != nil {
		log.Fatal("unable to create video listener", err)
	}

	manager.videoListener = listener

	// Increase the UDP receive buffer size
	// Default UDP buffer sizes vary on different operating systems
	bufferSize := 300000 // 300KB
	err = listener.SetReadBuffer(bufferSize)
	// if err != nil {
	// 	manager.logger.Warn().Err(err).Msg("unable to increase UDP buffer size")
	// }

	manager.videoTrack, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
		MimeType:    webrtc.MimeTypeH264,
		ClockRate:   90000,
		Channels:    0,
		SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f",
		RTCPFeedback: []webrtc.RTCPFeedback{
			{Type: "goog-remb", Parameter: ""},
			{Type: "ccm", Parameter: "fir"},
			{Type: "nack", Parameter: ""},
			{Type: "nack", Parameter: "pli"},
		}}, "vstream", "nistream")

	// if err != nil {
	// 	manager.logger.Panic().Err(err).Msg("unable to create video track")
	// }

	// ReadRTP reads RTP packets from the video listener
	go func() {
		inboundRTPPacket := make([]byte, 1600) // UDP MTU
		ssrc := uint32(0)
		seq := uint16(0)
		previousSeq := uint16(0)
		for {
			select {
			case <-manager.resetSequenceChan:
				seq = 0
				previousSeq = 0
				ssrc = 0
			default:
				n, _, err := manager.videoListener.ReadFrom(inboundRTPPacket)
				if err != nil {
					log.Println("failed to read RTP packet")
					continue
				}
				packet := &rtp.Packet{}

				if err := packet.Unmarshal(inboundRTPPacket[:n]); err != nil {
					log.Println("error during unmarshalling a packet: %s", err)
					continue
				}
				if ssrc == 0 {
					ssrc = packet.SSRC
				} else if packet.SSRC != ssrc && ssrc != 0 {
					if previousSeq > packet.SequenceNumber {
						// no point in sending this packet
						log.Println("dropping packet")
						continue
					}
					previousSeq = packet.SequenceNumber
					packet.SSRC = ssrc
					packet.SequenceNumber = seq
					packet.Header.SequenceNumber = seq
					seq++
				}

				if err = manager.videoTrack.WriteRTP(packet); err != nil {
					if errors.Is(err, io.ErrClosedPipe) {
						// The peerConnection has been closed.
						log.Println("peer connection closed")
						return
					}
					log.Println("failed to write RTP packet")
					continue
				}
			}
		}
	}()

	manager.started = true
}


// CreatePeer creates a new peer
func (manager *WebRTCManager) CreatePeer(id string) (error) {

	// reset the sequence number on every new peer
	manager.resetSequenceChan <- true

	configuration := webrtc.Configuration{
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}

	// Create new peer connection
	connection, err := manager.api.NewPeerConnection(configuration)
	if err != nil {
		return err
	}

	// For mouse move, scroll, clicks
	msNegotiated := true
	mouseDataChannel, err := connection.CreateDataChannel(
		"ms",
		&webrtc.DataChannelInit{
			Negotiated: &msNegotiated,
		},
	)
	if err != nil {
		return err
	}
	// For keystrokes
	kbNegotiated := true
	keyboardDataChannel, err := connection.CreateDataChannel(
		"kb",
		&webrtc.DataChannelInit{
			Negotiated: &kbNegotiated,
		},
	)
	if err != nil {
		return err
	}
	// For all other workspace events
	dataNegotiated := false
	appDataChannelOrdered := true
	appDataChannelMaxRetransmits := uint16(3)
	appDataChannel, err := connection.CreateDataChannel(
		"app",
		&webrtc.DataChannelInit{
			Negotiated:     &dataNegotiated,
			Ordered:        &appDataChannelOrdered,
			MaxRetransmits: &appDataChannelMaxRetransmits,
		},
	)
	if err != nil {
		return err
	}
	remoteDataChannelOrdered := true
	remoteDataChannelMaxRetransmits := uint16(3)
	remoteDataChannel, err := connection.CreateDataChannel(
		"rc",
		&webrtc.DataChannelInit{
			Ordered:        &remoteDataChannelOrdered,
			MaxRetransmits: &remoteDataChannelMaxRetransmits,
		},
	)

	if err != nil {
		return err
	}

	rtpVideo, err := connection.AddTrack(manager.videoTrack)
	if err != nil {
		return err
	}

	// rtpAudio, err := connection.AddTrack(manager.audioTrack)
	// if err != nil {
	// 	return nil, err
	// }

	connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateDisconnected:
			log.Println("peer disconnected")

		case webrtc.PeerConnectionStateFailed:
			log.Println("peer failed")

		case webrtc.PeerConnectionStateClosed:
			log.Println("peer closed")

		case webrtc.PeerConnectionStateConnected:
			log.Println("peer connected")

		case webrtc.PeerConnectionStateConnecting:
			log.Println("peer connecting")

		case webrtc.PeerConnectionStateNew:
			log.Println("peer new")
		}
	})

	connection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			manager.logger.Info().Msg("sent all ICECandidates")
			return
		}

		candidateString, err := json.Marshal(i.ToJSON())

		if err != nil {
			manager.logger.Warn().Err(err).Msg("converting ICECandidate to json failed")
			return
		}

		peer.AddServerICECandidates(string(candidateString))
	})

	connection.OnSignalingStateChange(func(state webrtc.SignalingState) {
		manager.logger.Info().
			Str("signalling-state", state.String()).
			Msg("Signalling state changed")
		switch state {
		case webrtc.SignalingStateClosed:
			peer.ResetIceCandidates()
		}
	})

	session.SetPeer(peer)

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpVideo.Read(rtcpBuf); rtcpErr != nil {
				if rtcpErr == io.EOF {
					manager.logger.Warn().Err(rtcpErr).Msg("RTCP connection closed")
					return
				}
				// reset the buffer
				rtcpBuf = rtcpBuf[:0]
				// skipping instead of returning as Read could fail during ffmpeg restart
				manager.logger.Warn().Err(rtcpErr).Msg("failed to read RTCP packet")
				continue
			}
		}
	}()

	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpAudio.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	return peer, nil

}
