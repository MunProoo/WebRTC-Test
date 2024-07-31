package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

// var iceServerProcessor *ICEServerProcessor
var config webrtc.Configuration

// nolint
var (
	// lock for peerConnections and trackLocals
	listLock sync.RWMutex
	// peerConnections []peerConnectionState
	peerConnections map[string]peerConnectionState
	trackLocals     map[string]TrackLocalRTP
)

func main() {
	config = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				// URLs: []string{"turn:211.207.68.244:3478"},
				URLs:           []string{"turn:192.168.30.186:3478"},
				Username:       "foo",
				Credential:     "bar",
				CredentialType: webrtc.ICECredentialTypePassword,
			},
		},
	}

	// iceServerProcessor = InitICEServer()

	peerConnections = map[string]peerConnectionState{}
	trackLocals = map[string]TrackLocalRTP{}

	http.HandleFunc("/ws", websocketHandler) // 웹소켓 (Client와 SDP 교환용)
	http.Handle("/", http.FileServer(http.Dir("./template/")))

	log.Println("httpServer started on :5000")

	// 모든 peer에게 3초마다 keyframe 요청
	go func() {
		for range time.NewTicker(time.Second * 3).C {
			dispatchKeyFrame()
		}
	}()

	log.Fatal(http.ListenAndServeTLS(":5000", "public.pem", "private.pem", nil))
}

// Handle incoming websockets
func websocketHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP request to Websocket
	unsafeConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	c := &threadSafeWriter{unsafeConn, sync.Mutex{}}
	defer c.Close() //nolint

	// Init to Peer's Info
	terminalID, terminalIDs := initInformation(c)

	// Create new PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Print(err)
		return
	}

	// When this frame returns close the PeerConnection
	defer peerConnection.Close() //nolint

	// Accept one audio and one video track incoming
	// 웹소켓 연결 시 peer의 미디어를 서버가 수신하기 위해 설정 (수신자)
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Print(err)
			return
		}
	}

	// dataChannel 생성
	dataChannel, err := peerConnection.CreateDataChannel("chat", nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Add our new PeerConnection to global list
	listLock.Lock()
	// peerConnections[peerID.String()] = peerConnectionState{peerConnection, c, dataChannel, false}
	peerConnections[terminalID] = peerConnectionState{peerConnection, c, dataChannel, false, terminalID}
	listLock.Unlock()

	// PeerConnection 콜백함수 설정
	PC_CallBackFunc(c, peerConnection, dataChannel, terminalIDs, terminalID)

	// Signal for the new PeerConnection
	SignalPeerConnections(terminalIDs, terminalID)

	// 웹소켓 수신
	message := &websocketMessage{}
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		} else if err := json.Unmarshal(raw, &message); err != nil {
			log.Println(err)
			return
		}

		handleWebSocketMessage(message, peerConnection)
	}
}
