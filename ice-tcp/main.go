// 시그널링 서버 <-> Peer 간의 통신만 구현되어있음. 그것도 TCP로

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pion/webrtc/v4"
)

var api *webrtc.API

func main() {
	settingEngine := webrtc.SettingEngine{}

	// UDP로만 연결
	settingEngine.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeUDP6,
	})

	fmt.Printf("Listening for ICE UDP")

	// Enable support only for TCP ICE candidates
	// settingEngine.SetNetworkTypes([]webrtc.NetworkType{
	// 	webrtc.NetworkTypeTCP4,
	// 	webrtc.NetworkTypeTCP6,
	// })

	// tcpListener, err := net.ListenTCP("tcp", &net.TCPAddr{
	// 	IP:   net.IP{0, 0, 0, 0},
	// 	Port: 8443,
	// })
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Printf("Listening for ICE TCP at %s\n", tcpListener.Addr())

	// // NewICETCPMux는 TCP연결만 사용하는 instance임. (ICE는 기본적으로 UDP를 사용한다.)
	// tcpMux := webrtc.NewICETCPMux(nil, tcpListener, 8)
	// settingEngine.SetICETCPMux(tcpMux)

	api = webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))

	http.Handle("/", http.FileServer(http.Dir(".")))

	http.HandleFunc("/doSignalling", doSignalling)

	fmt.Println("Open http://localhost:8080 to access this demo")
	panic(http.ListenAndServe(":8080", nil))
}

func doSignalling(w http.ResponseWriter, r *http.Request) {
	// ICE Candidate는 RTCPeerConnection 객체를 생성할 때 자동으로 수집된다.
	// webrtc.Configuration{}에 ICEServer를 설정하면 해당 서버를 통한다. (설정이 없다면 로컬 네트워크 인터페이스를 사용)
	// 로컬 네트워크 인터페이스면 로컬 네트워크에서만 동작함.
	// 폐쇄망의 경우에는 coturn이라는 오픈 소스를 통해 turn 서버를 사용할 수 있음. (STUN 기능 지원)
	// 인터넷이 되는 경우에는 google stun 서버 사용하면 됨

	// 로컬 후보: 로컬 네트워크 인터페이스(예: Wi-Fi, 이더넷)에서 후보를 수집합니다.
	// 리플렉티드 후보: STUN 서버를 통해 공용 IP 주소와 포트를 확인하여 수집합니다.
	// 릴레이 후보: TURN 서버를 통해 릴레이 주소를 사용하여 후보를 수집합니다.
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		panic(err)
	}

	// ICE 커넥션에 대해 상태 체크 (연결됨, 연결끊김)
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Send the current time via a DataChannel to the remote peer every 3 seconds
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnOpen(func() {
			for range time.Tick(time.Second * 3) {
				if err = d.SendText(time.Now().String()); err != nil {
					// 연결되기전에는 닫혀있으니까 해당 에러는 제외한듯
					if errors.Is(io.ErrClosedPipe, err) {
						return
					}
					panic(err)
				}
			}
		})
	})

	var offer webrtc.SessionDescription // SDP (Session Description Protocol)
	if err = json.NewDecoder(r.Body).Decode(&offer); err != nil {
		panic(err)
	}

	// 원격 Peer의 SDP 담기
	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	// ICE후보 수집이 완료되면 닫히는 채널 반환
	// ICE후보를 그냥 tricle하는 게 더 좋다. 왜냐면 연결 시작 시간이 길어질 수 있음.
	// Tricle : 점진적 전송. 일부 ICE후보를 먼저 전송해서 초기 연결 지연을 줄인다.
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// ICE후보 수집한 후에, 해당 정보를 포함하며, 보내려는 데이터를 answer에 담
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	} else if err = peerConnection.SetLocalDescription(answer); err != nil {
		// answer 생성이 되었으면. Local의 SDP 담는다
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	// 요약 : 후보 다 수집할 때까지 block
	<-gatherComplete

	response, err := json.Marshal(*peerConnection.LocalDescription())
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(response); err != nil {
		panic(err)
	}
}
