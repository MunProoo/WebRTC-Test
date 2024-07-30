package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

var iceServerProcessor *ICEServerProcessor
var config webrtc.Configuration

// nolint
var (
	// lock for peerConnections and trackLocals
	listLock sync.RWMutex
	// peerConnections []peerConnectionState
	peerConnections map[string]peerConnectionState
	trackLocals     map[string]TrackLocalRTP
)

type TrackLocalRTP struct {
	Track      *webrtc.TrackLocalStaticRTP
	TerminalID string
}

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

type chatMessage struct {
	Type     string `json:"type"`
	UserName string `json:"userName"`
	Message  string `json:"message"`
}

type peerConnectionState struct {
	peerConnection  *webrtc.PeerConnection
	websocket       *threadSafeWriter
	dataChannel     *webrtc.DataChannel
	dataChannelFlag bool // 데이터채널이 열린다음에 트랙 추가하도록 (메타데이터 전송의 이유때문에)
}

func main() {
	config = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs:           []string{"turn:192.168.30.186:3478"},
				Username:       "foo",
				Credential:     "bar",
				CredentialType: webrtc.ICECredentialTypePassword,
			},
		},
	}
	fmt.Println("ICE Server is Starting.... ")

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

// 새로운 트랙을 복사하여 서버내의 로컬 트랙으로 생성
func addTrack(t *webrtc.TrackRemote, terminalID string) *webrtc.TrackLocalStaticRTP {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		signalPeerConnections()
	}()

	fmt.Println(terminalID + "로부터 Stream 받는중")
	// Create a new TrackLocal with the same codec as our incoming
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	trackLocalRTP := TrackLocalRTP{Track: trackLocal, TerminalID: terminalID}

	trackLocals[t.ID()] = trackLocalRTP
	return trackLocal
}

func removeTrack(t *webrtc.TrackLocalStaticRTP) {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		signalPeerConnections()
	}()

	delete(trackLocals, t.ID())
}

// 트랙의 변화가 있으면 모든 peer에 대해 새로운 트랙 추가 or 트랙 제거
func signalPeerConnections() {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		dispatchKeyFrame()
	}()

	// 트랙 상태 동기화. 피어에게 새로운 offer 생성하여 전송
	attemptSync := func() (tryAgain bool) {
		// for i, _ := range peerConnections {
		for key, peerConnectionState := range peerConnections {
			// 연결 끊긴 peerConnection 제거
			// if peerConnections[i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
			// 	peerConnections = append(peerConnections[:i], peerConnections[i+1:]...)
			// 	return true // We modified the slice, start from the beginning
			// }
			if peerConnectionState.peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				delete(peerConnections, key)
			}

			// map of sender we already are sending, so we don't double send
			existingSenders := map[string]bool{}

			// // 송신자 : 서버 peer가 로컬 트랙을 원격 Peer로 전송
			// for _, sender := range peerConnections[i].peerConnection.GetSenders() {
			// 	if sender.Track() == nil {
			// 		continue
			// 	}

			// 	existingSenders[sender.Track().ID()] = true

			// 	// If we have a RTPSender that doesn't map to a existing track remove and signal
			// 	if _, ok := trackLocals[sender.Track().ID()]; !ok {
			// 		if err := peerConnections[i].peerConnection.RemoveTrack(sender); err != nil {
			// 			return true
			// 		}
			// 	}
			// }

			// // 수신자 : 원격 Peer로부터 미디어 트랙을 수신하는 객체
			// for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
			// 	if receiver.Track() == nil {
			// 		continue
			// 	}
			// 	// 동일한 트랙을 송신하거나 수신하는 루프백 방지
			// 	existingSenders[receiver.Track().ID()] = true
			// }

			// // Add all track we aren't sending yet to the PeerConnection
			// for trackID := range trackLocals {
			// 	if _, ok := existingSenders[trackID]; !ok {
			// 		if _, err := peerConnections[i].peerConnection.AddTrack(trackLocals[trackID]); err != nil {
			// 			return true
			// 		}
			// 	}
			// }

			// offer, err := peerConnections[i].peerConnection.CreateOffer(nil)
			// if err != nil {
			// 	return true
			// }

			// if err = peerConnections[i].peerConnection.SetLocalDescription(offer); err != nil {
			// 	return true
			// }

			// offerString, err := json.Marshal(offer)
			// if err != nil {
			// 	return true
			// }

			// if err = peerConnections[i].websocket.WriteJSON(&websocketMessage{
			// 	Event: "offer",
			// 	Data:  string(offerString),
			// }); err != nil {
			// 	return true
			// }

			// 송신자 : 서버 peer가 로컬 트랙을 원격 Peer로 전송
			for _, sender := range peerConnectionState.peerConnection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				// If we have a RTPSender that doesn't map to a existing track remove and signal
				if _, ok := trackLocals[sender.Track().ID()]; !ok {
					if err := peerConnectionState.peerConnection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			// 수신자 : 원격 Peer로부터 미디어 트랙을 수신하는 객체
			for _, receiver := range peerConnectionState.peerConnection.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}
				// 동일한 트랙을 송신하거나 수신하는 루프백 방지
				existingSenders[receiver.Track().ID()] = true
			}

			// Add all track we aren't sending yet to the PeerConnection
			for trackID, trackLocalRTP := range trackLocals {
				if _, ok := existingSenders[trackID]; !ok {

					// 트랙에 대한 Metadata 전송
					message := map[string]interface{}{
						"type":       "metadata",
						"terminalID": trackLocalRTP.TerminalID,
						"streamID":   trackLocalRTP.Track.StreamID(),
						"kind":       trackLocalRTP.Track.Kind().String(),
					}
					metaData, err := json.Marshal(message)
					if err != nil {
						log.Println(err)
						return
					}
					// 연결 끊기면 바로 connection, dataChannel 바로 삭제하니까 예외처리 해야함
					if _, ok := peerConnections[key]; ok {
						peerConnections[key].dataChannel.Send(metaData)
					}

					// 데이터채널 아직 수립 안됐음
					if !peerConnectionState.dataChannelFlag {
						break
					}

					// peer에 트랙 추가
					if _, err := peerConnectionState.peerConnection.AddTrack(trackLocalRTP.Track); err != nil {
						return true
					}

				}
			}

			offer, err := peerConnectionState.peerConnection.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = peerConnectionState.peerConnection.SetLocalDescription(offer); err != nil {
				return true
			}

			offerString, err := json.Marshal(offer)
			if err != nil {
				return true
			}

			if err = peerConnectionState.websocket.WriteJSON(&websocketMessage{
				Event: "offer",
				Data:  string(offerString),
			}); err != nil {
				return true
			}
		}

		return
	}

	// 25번의 동기화 시도가 실패하면 (Lock으로 인해 RemoveTrack과 AddTrack을 방해하고 있을 수도 있으므로) 3초 후 비동기적으로 다시 시도.
	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				signalPeerConnections()
			}()
			return
		}

		if !attemptSync() {
			// false 반환하면 동기화 성공 : 루프 종료
			// true 반환하면 다시 시도해야함
			break
		}
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
	// for i := range peerConnections {
	// 	for _, receiver := range peerConnections[i].peerConnection.GetReceivers() {
	// 		if receiver.Track() == nil {
	// 			continue
	// 		}

	// 		// 유효한 트랙에 대해 `PLI` RTCP 패킷을 전송. 이 패킷은 송신자에게 키 프레임을 요청함
	// 		_ = peerConnections[i].peerConnection.WriteRTCP([]rtcp.Packet{
	// 			&rtcp.PictureLossIndication{
	// 				MediaSSRC: uint32(receiver.Track().SSRC()), // SSRC (Syncronized Source)
	// 			},
	// 		})
	// 	}
	// }
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

	// When this frame returns close the Websocket
	defer c.Close() //nolint

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

	// chat용 정보
	peerID := uuid.New()

	// 출처 식별 (어디로부터 온 데이터인가!)
	terminalID := ""

	// Add our new PeerConnection to global list
	listLock.Lock()
	// peerConnections = append(peerConnections, peerConnectionState{peerConnection, c, dataChannel})
	peerConnections[peerID.String()] = peerConnectionState{peerConnection, c, dataChannel, false}
	listLock.Unlock()

	// Trickle ICE. Emit server candidate to client
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
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
	})

	// If PeerConnection is closed remove it from global list
	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				log.Print(err)
			}
		case webrtc.PeerConnectionStateClosed:
			signalPeerConnections()
		default:
		}
	})

	// OnTrack : 새로운 트랙이 수신될 때 호출
	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		// metaData 먼저 수신한 다음 진행하도록 로직 분리
		attemptToProcessTrack(t, terminalID, peerID.String())
	})

	// Handling dataChannel
	dataChannel.OnOpen(func() {
		fmt.Println("DataChannel Opened")
		dataChannel.SendText("Hello. It's sended from Server !")
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {

		var data chatMessage
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println(err)
		}

		peerConnectionState := peerConnections[peerID.String()]
		if !peerConnectionState.dataChannelFlag {
			peerConnectionState.dataChannelFlag = true
			peerConnections[peerID.String()] = peerConnectionState
		}

		terminalID = data.UserName

		fmt.Printf("Received Message : %s\n", data)
		broadCastDataChannelMessage(data, peerID.String())

	})

	// Signal for the new PeerConnection
	signalPeerConnections()

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

		switch message.Event {
		case "candidate":
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
				log.Println(err)
				return
			}

			if err := peerConnection.AddICECandidate(candidate); err != nil {
				log.Println(err)
				return
			}
		case "answer":
			answer := webrtc.SessionDescription{}
			if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
				log.Println(err)
				return
			}

			if err := peerConnection.SetRemoteDescription(answer); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

// Helper to make Gorilla Websockets threadsafe
type threadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

func (t *threadSafeWriter) WriteJSON(v interface{}) error {
	t.Lock()
	defer t.Unlock()

	return t.Conn.WriteJSON(v)
}

func broadCastDataChannelMessage(message chatMessage, peerID string) {
	for key, peerConnection := range peerConnections {
		if key == peerID {
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

// 데이터채널이 열렸으면 addTrack
// 안열렸으면 1초 후 재시도
func attemptToProcessTrack(t *webrtc.TrackRemote, terminalID string, peerID string) {

	if peerConnections[peerID].dataChannelFlag {
		// 수신된 새로운 트랙과 동일한 코덱을 가진 새로운 로컬 트랙을 만들어서 저장함
		trackLocal := addTrack(t, terminalID)
		defer removeTrack(trackLocal)

		buf := make([]byte, 1500)
		for {
			// 수신된 트랙의 데이터 읽기
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}

			// 수신된 트랙의 데이터를 새로운 로컬 트랙에 저장 (모든 peer에게 전송)
			if _, err = trackLocal.Write(buf[:i]); err != nil {
				return
			}
		}
	} else {
		// 1초 후 재시도
		go func() {
			time.Sleep(time.Second * 1)
			attemptToProcessTrack(t, terminalID, peerID)
		}()
		return
	}
}
