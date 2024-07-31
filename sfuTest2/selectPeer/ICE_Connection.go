// ICE Connection 관련 코드 분리
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

// PeerConnection의 콜백함수 설정
func PC_CallBackFunc(c *threadSafeWriter, peerConnection *webrtc.PeerConnection, dataChannel *webrtc.DataChannel, terminalIDs map[string]struct{}, terminalID string) {
	// Trickle ICE. Emit server candidate to client
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		PeerConnectionOnICECandidate(i, c)
	})

	// If PeerConnection is closed remove it from global list
	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				log.Print(err)
			}
		case webrtc.PeerConnectionStateClosed:
			SignalPeerConnections(terminalIDs, "") // broadCast
		default:
		}
	})

	// OnTrack : 새로운 트랙이 수신될 때 호출
	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		PeerConnectionOnTrack(t, terminalIDs, terminalID)
	})

	// Handling dataChannel
	dataChannel.OnOpen(func() {
		DataChannelOnOpen(dataChannel, terminalID)
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		DataChannelOnMessage(msg, terminalIDs, terminalID)
	})
}

// ICE 후보들 수집되는 경우
func PeerConnectionOnICECandidate(i *webrtc.ICECandidate, c *threadSafeWriter) {
	if i == nil {
		return
	}

	candidateString, err := json.Marshal(i.ToJSON())
	if err != nil {
		log.Println(err)
		return
	}

	if writeErr := c.WriteJSON(&websocketMessage{
		Event: "candidate",
		Data:  string(candidateString),
	}); writeErr != nil {
		log.Println(writeErr)
	}
}

// PeerConnection이 Track 수신하게 되면 수행
func PeerConnectionOnTrack(t *webrtc.TrackRemote, terminalIDs map[string]struct{}, terminalID string) {
	// 수신된 새로운 트랙과 동일한 코덱을 가진 새로운 로컬 트랙을 만들어서 저장함
	trackLocal := addTrack(t, terminalID)
	defer removeTrack(trackLocal, terminalIDs)

	buf := make([]byte, 1500)
	for {
		// 수신된 트랙의 데이터 읽기
		i, _, err := t.Read(buf)
		if err != nil {
			return
		}

		// 수신된 트랙의 데이터를 새로운 로컬 트랙에 저장 (모든 peer에게 전송할 수 있도록)
		if _, err = trackLocal.Write(buf[:i]); err != nil {
			return
		}
	}
}

func DataChannelOnOpen(dataChannel *webrtc.DataChannel, terminalID string) {
	// peer와 서버 연결되면, 연결되어있는 track list 전달
	fmt.Println("DataChannel Opened From : ", terminalID)

	data := makeTrackList()
	dataChannel.Send(data)

	peerConnectionState := peerConnections[terminalID]
	if !peerConnectionState.dataChannelFlag {
		peerConnectionState.dataChannelFlag = true
		peerConnections[terminalID] = peerConnectionState
	}
}

// DC 메시지 수신
func DataChannelOnMessage(msg webrtc.DataChannelMessage, terminalIDs map[string]struct{}, terminalID string) {

	var data chatMessage
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Println(err)
	}

	switch data.Type {
	// case "init":
	// 	terminalID = data.TerminalID
	// peerConnectionState := peerConnections[peerID.String()]
	// 	if !peerConnectionState.dataChannelFlag {
	// 		peerConnectionState.dataChannelFlag = true
	// 		peerConnections[peerID.String()] = peerConnectionState
	// 	}
	case "chat":
		fmt.Printf("Received Message : %s\n", data)
		broadCastDataChannelMessage(data, terminalID)
	case "trackOffer":
		clear(terminalIDs)
		for _, id := range data.Array {
			terminalIDs[id] = struct{}{} // 메모리 소모안하는 빈 구조체로 할당
		}
		SignalPeerConnections(terminalIDs, terminalID) // 선택한 트랙만 AddTrack
	default:
	}
}

// (채팅용) 데이터채널의 메시지 broadCast
// TODO : 필요 시 Room 개념 도입하여 각 Room에 참여한 peer에게만 전달
func broadCastDataChannelMessage(message chatMessage, terminalID string) {
	for key, peerConnection := range peerConnections {
		if key == terminalID {
			continue
		}
		data, err := json.Marshal(message)
		if err != nil {
			log.Println(err)
			return
		}
		peerConnection.dataChannel.Send(data)
	}
}

// dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call
// 모든 PeerConnection에 대해 키 프레임을 요청
// 목적 : 실시간 스트리밍에서 비디오 트랙의 상태를 유지하고, 새로운 피어가 연결될 때 비디오 스트림을 빠르게 동기화하도록 도움
func dispatchKeyFrame() {
	listLock.Lock()
	defer listLock.Unlock()

	for _, peerConnectionState := range peerConnections {
		for _, receiver := range peerConnectionState.peerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			// 유효한 트랙에 대해 `PLI` RTCP 패킷을 전송. 이 패킷은 송신자에게 키 프레임을 요청함
			_ = peerConnectionState.peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()), // SSRC (Syncronized Source)
				},
			})
		}
	}
}
