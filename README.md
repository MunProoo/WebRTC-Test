# WebRTC-Test

Go + [Pion WebRTC](https://github.com/pion/webrtc) 기반 **SFU(Selective Forwarding Unit)** 실험 프로젝트입니다.  
WebSocket 시그널링, 미디어 트랙 중계, DataChannel 채팅, 선택 Peer 시청·화면 공유까지 단계적으로 구현했습니다.

## 주요 기능

| 기능 | 설명 |
|------|------|
| **SFU 중계** | 클라이언트가 올린 A/V 트랙을 서버가 받아 다른 Peer에게 전달 |
| **WebSocket 시그널링** | Offer / Answer / ICE Candidate 교환 |
| **DataChannel 채팅** | 서버를 통한 메시지 fan-out (broadcast) |
| **선택 Peer 시청** | 원하는 Peer의 미디어만 구독 |
| **화면 공유** | Display capture 스트림 추가 (PC 브라우저 기준) |
| **Keyframe 요청** | 주기적 PLI/키프레임 요청으로 신규 참여자 화면 복구 |

## 아키텍처 개요

```
Browser A ──┐
            │  WebSocket (SDP / ICE)
Browser B ──┼──► SFU Server (Pion) ──► TrackLocalRTP 중계
            │         │
Browser C ──┘         └── DataChannel 채팅 broadcast
```

- **시그널링:** HTTPS + WebSocket (`/ws`)
- **미디어:** PeerConnection → 서버 TrackLocal → 다른 Peer로 Forward
- **채팅:** DataChannel 메시지를 서버가 수신·브로드캐스트

## 디렉터리 구조

```
WebRTC-Test/
├── SignallingServer/     # P2P용 시그널링 서버 실험
├── sfuTest/              # SFU 초기 프로토타입
├── sfuTest2/             # SFU 본 구현
│   ├── faceAndChatting/  # 화상 + 실시간 채팅
│   └── selectPeer/       # 화상 + 채팅 + 화면공유 + Peer 선택
├── ice-tcp/              # ICE / TCP 관련 실험
├── turnServer/           # TURN 서버 연동 참고
├── go.mod
└── README.md
```

포트폴리오·데모 기준으로는 **`sfuTest2/faceAndChatting`**, **`sfuTest2/selectPeer`** 를 보면 됩니다.

## 기술 스택

- **Language:** Go 1.21+
- **WebRTC:** `github.com/pion/webrtc/v4`
- **Signaling:** `github.com/gorilla/websocket`
- **Frontend:** Vanilla JS + HTML (브라우저 `getUserMedia` / `getDisplayMedia`)

## 실행 방법

### 사전 준비

1. Go 1.21 이상
2. 브라우저에서 `getUserMedia`를 쓰려면 **HTTPS**가 필요합니다. (로컬 `public.pem` / `private.pem` 사용)
3. NAT 환경이면 STUN/TURN을 `main.go`의 `ICEServers`에 맞게 수정하세요.

### 1) 화상 + 실시간 채팅 (`faceAndChatting`)

```bash
cd sfuTest2/faceAndChatting
go run .
```

1. 브라우저에서 `https://<본인IP>:5000` 접속  
2. 사용자 이름 입력 후 **채팅 시작**  
3. 여러 탭(또는 기기)으로 접속해 화상·채팅 확인  

### 2) 선택 Peer + 화면 공유 (`selectPeer`)

```bash
cd sfuTest2/selectPeer
go run .
```

1. 브라우저에서 `https://<본인IP>:5000` 접속  
2. 아이디 입력 후 **서버와 연결 시작**  
3. Peer 선택으로 특정 사용자 미디어만 수신, 필요 시 화면 공유  

> 화면 공유는 스마트폰에서 지원되지 않습니다.  
> 모바일과 같이 테스트할 때는 `index.html`의 `localDisplay`, `index.js`의 `displayStream` 관련 코드를 주석 처리하세요.

## 구현 포인트

- PeerConnection / Track / DataChannel 상태를 서버에서 관리하고, 참가·퇴장 시 트랙을 동기화합니다.
- 채팅은 DataChannel로 수신한 뒤 서버가 연결된 Peer에게 fan-out 합니다.
- `selectPeer`에서는 구독할 Peer를 골라 **필요한 미디어만** 전달받도록 했습니다.
- 신규 Peer 합류 시 화면이 깨지지 않도록 주기적으로 keyframe(PLI)을 요청합니다.

## 참고

- 본 저장소는 **학습·실험용** SFU 구현입니다. 프로덕션의 인증·권한·스케일아웃·녹화 파이프라인은 포함하지 않습니다.
- TURN 주소·계정 정보는 환경에 맞게 교체한 뒤 사용하세요.

## License

Personal / learning project.
