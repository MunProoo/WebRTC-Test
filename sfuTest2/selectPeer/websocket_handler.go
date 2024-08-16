// 웹소켓 관련 로직 분리
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Helper to make Gorilla Websockets threadsafe
type threadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

// 스레드 safe하게 웹소켓 메시지 입력
func (t *threadSafeWriter) WriteJSON(v interface{}) error {
	t.Lock()
	defer t.Unlock()

	return t.Conn.WriteJSON(v)
}

func initInformation(c *threadSafeWriter) (terminalID string, terminalIDs map[string]struct{}) {
	// InitMessage로 Peer가 누구인지 출처 확인. (단말기의 아이디)
	initMessage := map[string]interface{}{}
	if err := c.ReadJSON(&initMessage); err != nil {
		log.Print("read:", err)
	}

	// 출처 식별 (어디로부터 온 요청인가!)
	terminalID = initMessage["terminalID"].(string)

	// 트랙 요청중인 단말기의 리스트
	terminalIDs = map[string]struct{}{}

	return
}

func handleWebSocketMessage(message *websocketMessage, peerConnection *webrtc.PeerConnection) {
	// fmt.Println(message.Event, message.Data)
	switch message.Event {
	case "candidate":
		candidate := webrtc.ICECandidateInit{}
		if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
			fmt.Println(err)
			return
		}

		if err := peerConnection.AddICECandidate(candidate); err != nil {
			fmt.Println(err)
			return
		}
	case "answer":
		answer := webrtc.SessionDescription{}
		if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
			fmt.Println(err)
			return
		}

		if err := peerConnection.SetRemoteDescription(answer); err != nil {
			fmt.Println(err)
			return
		}
	}
}
