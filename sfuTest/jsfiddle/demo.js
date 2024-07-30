/* eslint-env browser */

// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Create peer conn
const pc = new RTCPeerConnection({
    // iceServers: [{
    //   urls: 'stun:stun.l.google.com:19302'
    // }]
  })
  
pc.oniceconnectionstatechange = (e) => {
  console.log('connection state change', pc.iceConnectionState)
}
pc.onicecandidate = (event) => {
  if (event.candidate === null) {
    document.getElementById('localSessionDescription').value = JSON.stringify(pc.localDescription)
    // document.getElementById('localSessionDescription').value = btoa(
    //   JSON.stringify(pc.localDescription)
    // )
  }
}

// pc.onnegotiationneeded = (e) => {
//   pc.createOffer()
//     .then(d => {
//       pc.setLocalDescription(d);

//       return fetch('/doSignalling', {
//         method:'post',
//         headers: {
//           'Accept': 'application/json, text/plain, */*',
//           'Content-Type': 'application/json'
//         },
//         body:JSON.stringify(d)
//       })// 시그널링 서버의 응답 처리 과정
//     })
//     .then(res => res.json())  // 시그널링 서버로부터 받은 응답을 JSON 형식으로 파싱
//     .then(res => {
//       document.getElementById('remoteSessionDescription').value = JSON.stringify(res)
//     })
//     .catch(console.error)
// }

    // ontrack : PC의 media 트랙이 추가될 때 발생하는 이벤트 처리
    // 상대 peer가 보내는 미디어 트랙을 수신할 때 호출됨 (영상 통화, 스트리밍)

    // onmessage : WebRTC 데이터 채널을 통해 수신된 메시지 처리
    // 텍스트 채팅, 파일 전송등을 수신함
pc.ontrack = (event) => {
  console.log('Got track event', event)
  const video = document.createElement('video')
  video.srcObject = event.streams[0]
  video.autoplay = true
  video.width = '500'
  const label = document.createElement('div')
  label.textContent = event.streams[0].id
  document.getElementById('serverVideos').appendChild(label)
  document.getElementById('serverVideos').appendChild(video)
}

navigator.mediaDevices
  .getUserMedia({
    video: {
      width: {
        ideal: 4096
      },
      height: {
        ideal: 2160
      },
      frameRate: {
        ideal: 60,
        min: 10
      }
    },
    audio: false
  })
  .then((stream) => {
    document.getElementById('browserVideo').srcObject = stream
    // 미디어 스트림 전송 (송신용)
    var transciever = pc.addTransceiver(stream.getVideoTracks()[0], {
      direction: 'sendonly',
      streams: [stream]
      // sendEncodings: [
      //   {
      //     rid: 'h',
      //     scaleResolutionDownBy: 4.0
      //   }
      // ]
    })

    console.log('transciever : ', transciever)
    // 미디어 스트림 수신용
    pc.addTransceiver('video')
    pc.addTransceiver('video')
    pc.addTransceiver('video')
  })

window.startSession = () => {
  const sd = document.getElementById('remoteSessionDescription').value
  if (sd === '') {
    return alert('Session Description must not be empty')
  }

  try {
    // console.log('answer', JSON.parse(atob(sd)))
    // pc.setRemoteDescription(JSON.parse(atob(sd))) // 편의상 base64 변형X
    pc.setRemoteDescription(JSON.parse(sd))
  } catch (e) {
    alert(e)
  }
}

window.copySDP = () => {
  const browserSDP = document.getElementById('localSessionDescription')

  browserSDP.focus()
  browserSDP.select()

  try {
    const successful = document.execCommand('copy')
    const msg = successful ? 'successful' : 'unsuccessful'
    console.log('Copying SDP was ' + msg)
  } catch (err) {
    console.log('Unable to copy SDP ' + err)
  }
}
function sendOffer() {
  pc.createOffer()
    .then(d => {
      pc.setLocalDescription(d);
      return fetch('/doSignalling', {
        method:'post',
        headers: {
          'Accept': 'application/json, text/plain, */*',
          'Content-Type': 'application/json'
        },
        body:JSON.stringify(d)
      })// 시그널링 서버의 응답 처리 과정
    })
    .then(res => res.json())  // 시그널링 서버로부터 받은 응답을 JSON 형식으로 파싱
    .then(res => {
      document.getElementById('remoteSessionDescription').value = JSON.stringify(res)
    })
    .catch(console.error)
}