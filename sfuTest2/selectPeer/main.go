package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
				// URLs: []string{"stun:stun.l.google.com:19302"},
				// URLs: []string{"turn:192.168.30.186:3478"},
				URLs: []string{
					"turn:192.168.30.186:8888?transport=udp",
					"turn:192.168.30.186:8888?transport=tcp",
				},
				// URLs: []string{
				// 	"turn:211.207.68.244:8888?transport=udp",
				// 	"turn:211.207.68.244:8888?transport=tcp",
				// },
				Username:       "foo",
				Credential:     "bar",
				CredentialType: webrtc.ICECredentialTypePassword,
			},
		},
	}

	// 새 로그 파일 생성 (기존 파일 덮어쓰기)
	logFile, err := os.Create("logfile.txt")
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	// 로그 출력 대상을 파일로 설정
	log.SetOutput(logFile)

	// iceServerProcessor = InitICEServer()

	peerConnections = map[string]peerConnectionState{}
	trackLocals = map[string]TrackLocalRTP{}

	http.HandleFunc("/ws", websocketHandler) // 웹소켓 (Client와 SDP 교환용)
	http.Handle("/", http.FileServer(http.Dir("./template/")))

	fmt.Println("httpServer started on :5000")

	// 모든 peer에게 3초마다 keyframe 요청
	go func() {
		for range time.NewTicker(time.Second * 3).C {
			dispatchKeyFrame()
		}
	}()

	// Port : 8888
	// go TurnServerProcessor()

	log.Fatal(http.ListenAndServeTLS(":5000", "public.pem", "private.pem", nil))
}

// Handle incoming websockets
func websocketHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("websocket req from: ", r.RemoteAddr)
	// Upgrade HTTP request to Websocket
	unsafeConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Print("upgrade:", err)
		return
	}

	c := &threadSafeWriter{unsafeConn, sync.Mutex{}}
	defer c.Close() //nolint

	// Init to Peer's Info
	// 단말 ID와는 별개로 srcAddr을 쓰든 해서 구분하는 게 나아보임
	terminalID, terminalIDs := initInformation(c)

	srcAddr := r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")]
	if _, ok := peerConnections[srcAddr]; ok {
		// 기존 연결이 있었다면, return
		// // 기존 트랙 삭제
		// removeLocalTrackAndReconnect(oldConnection)
		return
	}

	// Create new PeerConnection
	// 더 자세한 설정을 하기 위해선 Api를 통해서 Connection 만든다.
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		fmt.Print(err)
		return
	}

	// When this frame returns close the PeerConnection
	defer peerConnection.Close() //nolint

	// Add Receiver (display, webcam 등등)
	AddTrackReceiver(peerConnection)

	// dataChannel 생성
	dataChannel, err := peerConnection.CreateDataChannel("chat", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Add our new PeerConnection to global list
	listLock.Lock()
	peerConnectionState := peerConnectionState{peerConnection, c, dataChannel, false, terminalID, srcAddr}
	peerConnections[srcAddr] = peerConnectionState

	listLock.Unlock()

	fmt.Println("PeerConnection 생성. -단말ID : ", terminalID, "    -RemoteAddr : ", srcAddr)

	// PeerConnection 콜백함수 설정
	PC_CallBackFunc(peerConnectionState, terminalIDs)

	// Signal for the new PeerConnection
	SignalPeerConnections(terminalIDs, srcAddr)

	// 웹소켓 수신
	message := &websocketMessage{}
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		} else if err := json.Unmarshal(raw, &message); err != nil {
			fmt.Println(err)
			return
		}

		handleWebSocketMessage(message, peerConnection)
	}
}
