package main

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

func InitICEServer() *ICEServerProcessor {
	settingEngine := webrtc.SettingEngine{}

	settingEngine.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeUDP6,
		webrtc.NetworkTypeTCP4,
		webrtc.NetworkTypeTCP6,
	})

	api := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))
	PeerConnections := make(map[string]*webrtc.PeerConnection)

	iceServerProcessor := &ICEServerProcessor{
		Rooms:           make(map[string]*Room),
		Clients:         make(map[string]*Client),
		PeerConnections: PeerConnections,
		Api:             api,
	}

	return iceServerProcessor
}

func (iceServer *ICEServerProcessor) InitWebRTCConnection(conn *websocket.Conn, peerID string) {

	peerConnection, err := webrtc.NewPeerConnection(config)

	// peerConnection, err := iceServer.Api.NewPeerConnection(config)
	if err != nil {
		log.Println(err)
		panic(err)
	}

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		conn.WriteJSON(candidate.ToJSON())
	})

	outputTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, peerID, "stream_from_server")
	if err != nil {
		log.Println(err)
		panic(err)
	}
	iceServer.OutputTracks[peerID] = outputTrack

	// 현재 서버와 연결하는 peer의 로컬 스트림을 위한 트랜시버 추가
	if _, err = peerConnection.AddTransceiverFromTrack(outputTrack, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionSendonly,
	}); err != nil {
		log.Println(err)
		panic(err)
	}

	/*
		나중에 추가될 다른 peer의 스트림을 위한 트랜시버 추가 (미리 만들어 놓는 방법)
		Track이 들어올 때마다 동적으로 AddTransceiverFromTrack을 해주면 된다.
	*/
	// if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{
	// 	Direction: webrtc.RTPTransceiverDirectionRecvonly,
	// }); err != nil {
	// 	log.Println(err)
	// 	panic(err)
	// }

	// ICE 커넥션 상태 체크 (ice 커넥션이 P2P 통신을 말하는 건가)
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// offer 생성
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		log.Println(err)
		panic(err)
	}

	// ICE 후보 수집 먼저 해야 SDP에 해당 정보 담아서 보냄
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	if err = peerConnection.SetLocalDescription(offer); err != nil {
		log.Println(err)
		panic(err)
	}
	// 수집완료 후 전송
	<-gatherComplete

	// offer 전송
	if err = conn.WriteJSON(map[string]interface{}{
		"type": "offer",
		"sdp":  peerConnection.LocalDescription().SDP,
	}); err != nil {
		log.Println(err)
	}

	// answer 수신
	var answer webrtc.SessionDescription
	if err = conn.ReadJSON(&answer); err != nil {
		log.Println(err)
	}

	if err = peerConnection.SetRemoteDescription(answer); err != nil {
		log.Println(err)
		panic(err)
	}

	// 수신된 ICE 후보를 처리하는 루프
	go func() {
		for {
			var candidate webrtc.ICECandidateInit
			if err := conn.ReadJSON(&candidate); err != nil {
				log.Println(err)
				return
			}
			if err := peerConnection.AddICECandidate(candidate); err != nil {
				log.Println(err)
				return
			}
		}
	}()

	// -------------------------------------------------------------------------------------------------

	// 채팅, 파일전송등을 위한 데이터채널

	iceServer.PeerConnections[peerID] = peerConnection
}
