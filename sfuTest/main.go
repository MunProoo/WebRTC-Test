// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js
// +build !js

// simulcast demonstrates of how to handle incoming track with multiple simulcast rtp streams and show all them back.
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

var PeerConnections map[string]*webrtc.PeerConnection

func main() {

	settingEngine := webrtc.SettingEngine{}

	// UDP로만 연결
	settingEngine.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeUDP6,
	})

	api = webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))

	PeerConnections = make(map[string]*webrtc.PeerConnection)

	fmt.Printf("Listening for ICE UDP")

	// Everything below is the Pion WebRTC API! Thanks for using it ❤️.

	// Prepare the configuration
	// config := webrtc.Configuration{
	// 	ICEServers: []webrtc.ICEServer{
	// 		{
	// 			URLs: []string{"stun:stun.l.google.com:19302"},
	// 		},
	// 	},
	// }

	// // Create a new RTCPeerConnection
	// peerConnection, err := webrtc.NewPeerConnection(config)
	// if err != nil {
	// 	panic(err)
	// }
	// defer func() {
	// 	if cErr := peerConnection.Close(); cErr != nil {
	// 		fmt.Printf("cannot close peerConnection: %v\n", cErr)
	// 	}
	// }()

	// outputTracks := map[string]*webrtc.TrackLocalStaticRTP{}

	// // Create Track that we send video back to browser on
	// outputTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video_q", "pion_q")
	// if err != nil {
	// 	panic(err)
	// }
	// outputTracks["q"] = outputTrack

	// outputTrack, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video_h", "pion_h")
	// if err != nil {
	// 	panic(err)
	// }
	// outputTracks["h"] = outputTrack

	// outputTrack, err = webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video_f", "pion_f")
	// if err != nil {
	// 	panic(err)
	// }
	// outputTracks["f"] = outputTrack

	// if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
	// 	panic(err)
	// }

	// // Add this newly created track to the PeerConnection to send back video
	// if _, err = peerConnection.AddTransceiverFromTrack(outputTracks["q"], webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly}); err != nil {
	// 	panic(err)
	// }
	// if _, err = peerConnection.AddTransceiverFromTrack(outputTracks["h"], webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly}); err != nil {
	// 	panic(err)
	// }
	// if _, err = peerConnection.AddTransceiverFromTrack(outputTracks["f"], webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly}); err != nil {
	// 	panic(err)
	// }

	// // Read incoming RTCP packets
	// // Before these packets are returned they are processed by interceptors. For things
	// // like NACK this needs to be called.
	// processRTCP := func(rtpSender *webrtc.RTPSender) {
	// 	rtcpBuf := make([]byte, 1500)
	// 	for {
	// 		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
	// 			return
	// 		}
	// 	}
	// }
	// for _, rtpSender := range peerConnection.GetSenders() {
	// 	go processRTCP(rtpSender)
	// }

	// Wait for the offer to be pasted
	// offer := webrtc.SessionDescription{}
	// decode(readFile(), &offer)
	// decode(readUntilNewline(), &offer)

	// if err = peerConnection.SetRemoteDescription(offer); err != nil {
	// 	panic(err)
	// }

	// Set a handler for when a new remote track starts
	// peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) { //nolint: revive
	// 	fmt.Println("Track has started")

	// 	// Start reading from all the streams and sending them to the related output track
	// 	rid := track.RID()

	// 	// 3초마다 PLI (Picture Loss Indication) 패킷 전송하는 고루틴 작성
	// 	go func() {
	// 		ticker := time.NewTicker(3 * time.Second)
	// 		for range ticker.C {
	// 			fmt.Printf("Sending pli for stream with rid: %q, ssrc: %d\n", track.RID(), track.SSRC())
	// 			if writeErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}}); writeErr != nil {
	// 				fmt.Println(writeErr)
	// 			}
	// 		}
	// 	}()
	// 	go func() { // 'f', 'q', 'h' 모든 트랙 받기 위해서 고루틴으로 변경
	// 		for {
	// 			// Read RTP packets being sent to Pion
	// 			packet, _, readErr := track.ReadRTP()
	// 			if readErr != nil {
	// 				panic(readErr)
	// 			}

	// 			if writeErr := outputTracks[rid].WriteRTP(packet); writeErr != nil && !errors.Is(writeErr, io.ErrClosedPipe) {
	// 				panic(writeErr)
	// 			}
	// 		}
	// 	}()
	// })

	// // Set the handler for Peer connection state
	// // This will notify you when the peer has connected/disconnected
	// peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
	// 	fmt.Printf("Peer Connection State has changed: %s\n", s.String())

	// 	if s == webrtc.PeerConnectionStateFailed {
	// 		// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
	// 		// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
	// 		// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
	// 		fmt.Println("Peer Connection has gone to failed exiting")
	// 		os.Exit(0)
	// 	}

	// 	if s == webrtc.PeerConnectionStateClosed {
	// 		// PeerConnection was explicitly closed. This usually happens from a DTLS CloseNotify
	// 		fmt.Println("Peer Connection has gone to closed exiting")
	// 		os.Exit(0)
	// 	}
	// })

	// // Create an answer
	// answer, err := peerConnection.CreateAnswer(nil)
	// if err != nil {
	// 	panic(err)
	// }

	// // Create channel that is blocked until ICE Gathering is complete
	// gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// // Sets the LocalDescription, and starts our UDP listeners
	// err = peerConnection.SetLocalDescription(answer)
	// if err != nil {
	// 	panic(err)
	// }

	// // Block until ICE Gathering is complete, disabling trickle ICE
	// // we do this because we only can exchange one signaling message
	// // in a production application you should exchange ICE Candidates via OnICECandidate
	// <-gatherComplete

	// // Output the answer in base64 so we can paste it in browser
	// fmt.Println(encode(peerConnection.LocalDescription()))

	// Block forever
	// select {}

	http.Handle("/", http.FileServer(http.Dir("./jsfiddle")))

	http.HandleFunc("/doSignalling", doSignalling)

	fmt.Println("Open https://localhost:8080 to access this demo")
	panic(http.ListenAndServeTLS("0.0.0.0:8080", "public.pem", "private.pem", nil))
	// panic(http.ListenAndServe("0.0.0.0:8080", nil))
}

// Read from stdin until we get a newline
func readUntilNewline() (in string) {
	var err error

	r := bufio.NewReader(os.Stdin)
	for {
		in, err = r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			panic(err)
		}

		if in = strings.TrimSpace(in); len(in) > 0 {
			break
		}
	}

	fmt.Println("")
	return
}

func readFile() (in string) {
	file, err := os.Open("my_file.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	r := bufio.NewReader(file)

	for {
		in, err = r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			panic(err)
		}

		if in = strings.TrimSpace(in); len(in) > 0 {
			break
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	fmt.Println("")
	return
}

// JSON encode + base64 a SessionDescription
func encode(obj *webrtc.SessionDescription) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(b)
}

// Decode a base64 and unmarshal JSON into a SessionDescription
func decode(in string, obj *webrtc.SessionDescription) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(b, obj); err != nil {
		panic(err)
	}
}

var api *webrtc.API

func doSignalling(w http.ResponseWriter, r *http.Request) {

	// ab := r.Header.Get("User-Agent")
	// fmt.Println(ab)
	fmt.Println("요청 들어옴")

	config := webrtc.Configuration{
		// ICEServers: []webrtc.ICEServer{
		// 	{
		// 		URLs: []string{"stun:stun.l.google.com:19302"},
		// 	},
		// },
	}

	// ICE Candidate는 RTCPeerConnection 객체를 생성할 때 자동으로 수집된다.
	// webrtc.Configuration{}에 ICEServer를 설정하면 해당 서버를 통한다. (설정이 없다면 로컬 네트워크 인터페이스를 사용)
	// 로컬 네트워크 인터페이스면 로컬 네트워크에서만 동작함.
	// 폐쇄망의 경우에는 coturn이라는 오픈 소스를 통해 turn 서버를 사용할 수 있음. (STUN 기능 지원)
	// 인터넷이 되는 경우에는 google stun 서버 사용하면 됨

	// 로컬 후보: 로컬 네트워크 인터페이스(예: Wi-Fi, 이더넷)에서 후보를 수집합니다.
	// 리플렉티드 후보: STUN 서버를 통해 공용 IP 주소와 포트를 확인하여 수집합니다.
	// 릴레이 후보: TURN 서버를 통해 릴레이 주소를 사용하여 후보를 수집합니다.
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	id := r.RemoteAddr
	PeerConnections[id] = peerConnection

	// Create a new RTCPeerConnection
	// defer func() {
	// 	if cErr := peerConnection.Close(); cErr != nil {
	// 		fmt.Printf("cannot close peerConnection: %v\n", cErr)
	// 	}
	// }()

	// 이건 필요 없겠는걸. Peer마다 Track을 1개씩 가지고 있다치면 그 Track을 꺼내서 쓰니까..?
	outputTracks := map[string]*webrtc.TrackLocalStaticRTP{}

	// Create Track that we send video back to browser on
	outputTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video_h", "pion_h")
	if err != nil {
		panic(err)
	}
	outputTracks["h"] = outputTrack

	// RTP트랜시버 : 미디어 송수신을 위한 인터페이스
	// 비디오 수신 전용(recvOnly) 트랜시버를 추가
	if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		panic(err)
	}

	// 비디오 트랙을 peer에게 보낼 수 있도록 송신 전용 트랜시버 추가
	if _, err = peerConnection.AddTransceiverFromTrack(outputTracks["h"], webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly}); err != nil {
		panic(err)
	}

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.

	// processRTCP := func(rtpSender *webrtc.RTPSender) {
	// 	rtcpBuf := make([]byte, 1500)
	// 	for {
	// 		if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
	// 			return
	// 		}
	// 	}
	// }
	// for _, rtpSender := range peerConnection.GetSenders() {
	// 	go processRTCP(rtpSender)
	// }

	// ICE 커넥션에 대해 상태 체크 (연결됨, 연결끊김)
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Send the current time via a DataChannel to the remote peer every 3 seconds
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnOpen(func() {
			for range time.Tick(time.Second * 3) {
				if err = d.SendText(time.Now().String()); err != nil {
					// 연결되기전에는 닫혀있으니까 해당 상황은 제외
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

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) { //nolint: revive
		fmt.Println("Track has started")

		// Start reading from all the streams and sending them to the related output track
		rid := track.RID()
		fmt.Printf("Received track with RID: %s\n", rid)

		for _, pc := range PeerConnections {
			if pc != peerConnection {
				// pc에 track 생성
				pcTrack, err := webrtc.NewTrackLocalStaticRTP(track.Codec().RTPCodecCapability, track.ID(), "another Peer")
				if err != nil {
					panic(err)
				}

				// Add the track to other PeerConnections
				rtpSender, err := pc.AddTrack(pcTrack)
				if err != nil {
					panic(err)
				}

				go func() {
					for {
						packet, _, readErr := track.ReadRTP()
						if readErr != nil {
							return
						}

						// connection에 스트림(rtp 패킷) 전달
						if writeErr := pcTrack.WriteRTP(packet); writeErr != nil {
							return
						}
					}
				}()

				go func() {
					// rtcp 패킷 체크 : 스트림 상태 모니터링 및 제어
					rtcpBuf := make([]byte, 1500)
					for {
						if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
							return
						}
					}
				}()
			}
		}

		// 3초마다 PLI (Picture Loss Indication) 패킷 전송하는 고루틴 작성
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			for range ticker.C {
				fmt.Printf("Sending pli for stream with rid: %q, ssrc: %d\n", track.RID(), track.SSRC())
				if writeErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}}); writeErr != nil {
					fmt.Println(writeErr)
				}
			}
		}()
		go func() {
			packetChan := make(chan *rtp.Packet, 100) // 버퍼 크기 조정 가능
			go func() {
				for packet := range packetChan {
					if writeErr := outputTrack.WriteRTP(packet); writeErr != nil && !errors.Is(writeErr, io.ErrClosedPipe) {
						fmt.Println(writeErr)
					}
				}
			}()
			for {
				packet, _, readErr := track.ReadRTP()
				if readErr != nil {
					close(packetChan)
					return
				}
				packetChan <- packet
			}
		}()
	})

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			fmt.Println("Peer Connection has gone to failed exiting")
			// os.Exit(0)
			// reconnect logic here!!
		}

		if s == webrtc.PeerConnectionStateClosed {
			// PeerConnection was explicitly closed. This usually happens from a DTLS CloseNotify
			fmt.Println("Peer Connection has gone to closed exiting")
			// os.Exit(0)
		}
	})

	// Create an answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	answer.SDP = strings.Replace(answer.SDP, "useinbandfec=1", "useinbandfec=1; max-fs=120; max-fr=60", -1) // 예: 최대 프레임 크기 및 프레임 레이트 조정

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	// Output the answer in base64 so we can paste it in browser
	fmt.Println(encode(peerConnection.LocalDescription()))
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
