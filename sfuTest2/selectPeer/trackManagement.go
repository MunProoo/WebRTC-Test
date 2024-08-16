// 미디어 트랙 관리 로직 분리 (peer의 로컬 트랙 받기, 서버의 트랙 관리, 원격 트랙 추가)
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pion/webrtc/v4"
)

// 새로운 트랙을 복사하여 서버내의 로컬 트랙으로 생성
func addTrack(t *webrtc.TrackRemote, terminalID string) *webrtc.TrackLocalStaticRTP {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		// signalPeerConnections() // Peer들에게 반영
	}()

	fmt.Println(terminalID + "로부터 Stream 받는중")
	// Create a new TrackLocal with the same codec as our incoming
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	// trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, terminalID, t.StreamID())
	if err != nil {
		panic(err)
	}

	trackLocalRTP := TrackLocalRTP{Track: trackLocal, TerminalID: terminalID}

	trackLocals[t.ID()] = trackLocalRTP

	// Track이 추가되었으니 알려준다. (Audio는 제외)
	for _, pcState := range peerConnections {
		if pcState.dataChannelFlag && t.Kind().String() == "video" {
			fmt.Println("track added from : ", terminalID)
			// TrackList 전달
			data := makeTrackList()
			pcState.dataChannel.Send(data)

			// PeerList 전달
			data = makePeerList()
			pcState.dataChannel.Send(data)
		}
	}

	return trackLocal
}

func removeTrack(t *webrtc.TrackLocalStaticRTP, terminalIDs map[string]struct{}) {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		SignalPeerConnections(terminalIDs, "")
	}()

	delete(trackLocals, t.ID())

	// Track이 삭제되었으니 알려준다.
	for _, pcState := range peerConnections {
		if pcState.dataChannelFlag {
			// TrackList 전달
			data := makeTrackList()
			pcState.dataChannel.Send(data)

			// PeerList 전달
			data = makePeerList()
			pcState.dataChannel.Send(data)
		}
	}
}

/*
	Peer 연결, 상태 동기화 및 트랙 관리.

Peer로부터 트랙요청을 받고 트랙을 추가, 삭제해준다. -> Peer 단일 동작
갖고있던 트랙의 연결이 끊기면, 송신중이던 트랙을 없앤다. -> broadCast 동작
*/
func SignalPeerConnections(terminalIDs map[string]struct{}, peerID string) {
	listLock.Lock()
	defer func() {
		listLock.Unlock()
		dispatchKeyFrame()
	}()

	fmt.Println(peerID + " 트랙 동기화 중")

	// 트랙 상태 동기화. 피어에게 새로운 offer 생성하여 전송
	attemptSync := func(peerID string) (tryAgain bool) {
		if peerID == "" { // broadCast
			for key, pcs := range peerConnections {
				return pcs.TrackManagement(key, terminalIDs)
			}
		} else {
			pcs := peerConnections[peerID]
			return pcs.TrackManagement(peerID, terminalIDs)
		}

		return
	}

	// 25번의 동기화 시도가 실패하면 (Lock으로 인해 RemoveTrack과 AddTrack을 방해하고 있을 수도 있으므로) 3초 후 비동기적으로 다시 시도.
	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				SignalPeerConnections(terminalIDs, peerID)
			}()
			return
		}

		if !attemptSync(peerID) {
			// false 반환하면 동기화 성공 : 루프 종료
			// true 반환하면 다시 시도해야함
			break
		}
	}
}

/*
Peer에게 송신하는 트랙 관리
key : peer 식별자
terminalIDs : peer가 요청한 단말기
*/
func (pcs *peerConnectionState) TrackManagement(key string, terminalIDs map[string]struct{}) bool {
	// ICE 연결 끊긴 peerConnection 제거
	if pcs.peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
		delete(peerConnections, key)
	}

	// map of sender we already are sending, so we don't double send
	existingSenders := map[string]bool{}

	// 송신자 : 서버가 로컬 트랙을 원격 Peer로 전송
	// 이미 보내고 있는 Track의 ID 체크
	for _, sender := range pcs.peerConnection.GetSenders() {
		if sender.Track() == nil {
			continue
		}
		// 동일한 트랙을 수신하지 않도록 관리
		existingSenders[sender.Track().ID()] = true

		// ICE 연결 끊긴 Peer의 트랙 제거
		if trackLocalRTP, ok := trackLocals[sender.Track().ID()]; !ok {
			if err := pcs.peerConnection.RemoveTrack(sender); err != nil {
				return true
			}
		} else if _, ok := terminalIDs[trackLocalRTP.TerminalID]; !ok {
			// 선택한 단말이 아니면 트랙 제거
			if err := pcs.peerConnection.RemoveTrack(sender); err != nil {
				return true
			}
		}
	}

	// 수신자 : 원격 Peer로부터 미디어 트랙을 수신하는 객체
	for _, receiver := range pcs.peerConnection.GetReceivers() {
		if receiver.Track() == nil {
			continue
		}
		// 본인의 트랙을 송신하지 않도록 관리
		// existingSenders[receiver.Track().ID()] = true
	}

	// Add all track we aren't sending yet to the PeerConnection
	for trackID, trackLocalRTP := range trackLocals {
		if _, ok := existingSenders[trackID]; !ok {
			if _, ok2 := terminalIDs[trackLocalRTP.TerminalID]; ok2 { // 선택한 단말만 트랙 추가해주도록
				// 트랙에 대한 Metadata 전송
				message := map[string]interface{}{
					"type":       "metadata",
					"terminalID": trackLocalRTP.TerminalID,
					"streamID":   trackLocalRTP.Track.StreamID(),
					"kind":       trackLocalRTP.Track.Kind().String(),
				}
				metaData, err := json.Marshal(message)
				if err != nil {
					fmt.Println(err)
					return false
				}
				// ICE 연결 끊기면 바로 connection, dataChannel 바로 삭제하니까 예외처리 해야함
				if _, ok := peerConnections[key]; ok {
					peerConnections[key].dataChannel.Send(metaData)
				}

				// 데이터채널 아직 수립 안됐음 -> 메타데이터 전송을 위해 다시 시도해야함
				if !pcs.dataChannelFlag {
					break
				}

				// peer에 트랙 추가
				if _, err := pcs.peerConnection.AddTrack(trackLocalRTP.Track); err != nil {
					return true
				}
			}
		}
	}

	offer, err := pcs.peerConnection.CreateOffer(nil)
	if err != nil {
		return true
	}

	if err = pcs.peerConnection.SetLocalDescription(offer); err != nil {
		return true
	}

	offerString, err := json.Marshal(offer)
	if err != nil {
		return true
	}

	if err = pcs.websocket.WriteJSON(&websocketMessage{
		Event: "offer",
		Data:  string(offerString),
	}); err != nil {
		return true
	}
	return false
}

// 기존 트랙을 삭제하고, 해당 트랙을 가지고 있던 Connection에 새로 offer를 날려 동기화한다.
// 재연결 시도를 할 경우, 기존에 Peer가 수신하고있던 트랙을 없애버리기 위함.!
func removeLocalTrackAndReconnect(oldPcs peerConnectionState) {
	// 기존 트랙 삭제
	for _, receiver := range oldPcs.peerConnection.GetReceivers() {
		if receiver.Tracks() == nil {
			continue
		}
		delete(trackLocals, receiver.Track().ID())
	}

	for _, pcs := range peerConnections {
		pcs.removeOffer()
	}

}

func (pcs *peerConnectionState) removeOffer() {
	// 송신자 : 서버가 로컬 트랙을 원격 Peer로 전송
	// 이미 보내고 있는 Track의 ID 체크
	for _, sender := range pcs.peerConnection.GetSenders() {
		if sender.Track() == nil {
			continue
		}

		// ICE 연결 끊긴 Peer의 트랙 제거
		if _, ok := trackLocals[sender.Track().ID()]; !ok {
			if err := pcs.peerConnection.RemoveTrack(sender); err != nil {
				return
			}
		}
	}

	offer, err := pcs.peerConnection.CreateOffer(nil)
	if err != nil {
		return
	}

	if err = pcs.peerConnection.SetLocalDescription(offer); err != nil {
		return
	}

	offerString, err := json.Marshal(offer)
	if err != nil {
		return
	}

	if err = pcs.websocket.WriteJSON(&websocketMessage{
		Event: "offer",
		Data:  string(offerString),
	}); err != nil {
		fmt.Println(err)
		return
	}
}
