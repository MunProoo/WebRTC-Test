// 구조체 정의 분리
// 모듈화를 하는게 맞지만, 편의상 정리없이 package main 하나로
package main

import "github.com/pion/webrtc/v4"

type TrackLocalRTP struct {
	Track      *webrtc.TrackLocalStaticRTP
	TerminalID string
}

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

type dataChannelMessage struct {
	Type       string   `json:"type"`
	TerminalID string   `json:"terminalID"` // 본인 아이디
	Message    string   `json:"message"`
	Array      []string `json:"array"`
	CallerID   string   `json:"callerID"`
	ReceiverID string   `json:"receiverID"`
}

type peerConnectionState struct {
	peerConnection  *webrtc.PeerConnection
	websocket       *threadSafeWriter
	dataChannel     *webrtc.DataChannel
	dataChannelFlag bool // 데이터채널이 열린다음에 트랙 추가하도록 (메타데이터 전송의 이유때문에)
	terminalID      string
	srcAddr         string // 요청 IP
}
