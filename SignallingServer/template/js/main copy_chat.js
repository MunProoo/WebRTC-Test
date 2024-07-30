'use strict';

import CustomPeer from "./customPeer.js"; // 여러 Peer 정보 저장용

const peers = new Map();

var isChannelReady = false;
var offerFlag = false;  // join한 peer는 offer 까지 보내도록. (그 외는 PeerConnection만 생성)
var localStream;
var pc;
var remoteStream;
var turnReady;

var mediaFlag = false; // 웹캠을 사용할 수 있다 없다.

var clientID;
var dataChannel;

var pcConfig = {
  'iceServers': [{
    'urls': 'turn:192.168.30.186:3478',
    'username':'foo',
    'credential' :'bar'
    // 'urls': 'stun:stun.l.google.com:19302'
  }]
};

// Set up audio and video regardless of what devices are present.
var sdpConstraints = {
  offerToReceiveAudio: true,
  offerToReceiveVideo: true
};

/////////////////////////////////////////////

var room = document.getElementById('roomName').value;
// Could prompt for room name:
// room = prompt('Enter room name:');

// var socket = io.connect();
var address = window.location.host;
var socket = new WebSocket('wss://'+address+'/ws');

navigator.mediaDevices.getUserMedia({
  audio: false,
  video: true
})
.then(gotStream)
.catch(function(e) {
  console.log('getUserMedia() error: ' + e.name);
});

socket.onopen = function() {
  console.log("websocket connection opened");
}

function joinRoom() {
  if (typeof localStream !== 'undefined') {
    room = document.getElementById('roomName').value;

    console.log(JSON.stringify({type:'create or join', room:room}));
    socket.send(JSON.stringify({type:'create or join', room:room}));
    // sendMessage('got user media');
  } else {
    alert("당신의 웹캠. 안나오고 있다.");
  }
}

socket.onmessage = function(event) {
  var message = JSON.parse(event.data);
  console.log('Client received message: ', message);

  switch(message.type) {
    case 'message':
      handleMessage(message);
      break;
    case 'created':
      console.log('Created room name : ' + message.room);
      // isInitiator = true;

      // connectFlag = true;
      break;
    case 'full':
      console.log('Room ' + message.room + ' is full');
      alert("방이 꽉 찼습니다.");
      break;
    case 'knock':
      console.error(message.client+' made a request to join room ' + room + '!');
      isChannelReady = true;
      
      // connectFlag = true;
      if(mediaFlag) {
        maybeStart(message.client);
      }
      break;
    case 'joined':
      console.error('Successfully Entered the room : ' + room);
      isChannelReady = true;

      offerFlag = true;
      if(mediaFlag) {
        maybeStart(message.client);
      }
      break;
    case 'log':
      console.log.apply(console, array);
      break;
    case 'offer': // P2P 연결 요청 들어옴
      // 이 부분은 필요 없을 듯
      // if(!connectFlag) { 
      //   maybeStart();
      // }
      pc.setRemoteDescription(new RTCSessionDescription(message));
      doAnswer();
      break;
    case 'answer': // P2P 연결 요청 응답옴
      pc.setRemoteDescription(new RTCSessionDescription(message));
      break;
    case 'candidate':
      var candidate = new RTCIceCandidate({
        sdpMLineIndex : message.label,
        candidate : message.candidate
      });
      pc.addIceCandidate(candidate);
      break;
    case 'bye':
      handleRemoteHangup();
  }

}

function handleMessage(message) {
  console.log('Client received message from : ' + message.client);

  clientID = message.client;
  var content = message.content
  if(content == 'got user media') {
    // maybeStart()
    console.error("나의 미디어가 잘 준비되었다는 것을 확인했음.")
  }
}

function sendMessage(message) {
  console.log('Client sending message: ', message);
  socket.send(JSON.stringify(message));
}

var localVideo = document.querySelector('#localVideo');
var remoteVideo = document.querySelector('#remoteVideo'); 

function gotStream(stream) {
  console.log('Adding local stream.');
  localStream = stream;
  localVideo.srcObject = stream;

  // sendMessage('got user media');
  // if (isInitiator) {
  //   maybeStart();
  // }
  mediaFlag = true; 
}

var constraints = {
  video: true
};

// console.log('Getting user media with constraints', constraints);

// if (location.hostname !== 'localhost') {
//   requestTurn(
//     'https://computeengineondemand.appspot.com/turn?username=41784574&key=4080218913'
//   );
// }

function maybeStart(peerID) {
  if (typeof localStream !== 'undefined' && isChannelReady) {
    console.log('>>>>>> creating peer connection ', clientID);
    createPeerConnection(peerID);

    // pc.addStream(localStream);
    pc.addTrack(localStream.getTracks()[0], localStream);

    if (offerFlag) {
      doCall();
    }
  }
}

window.onbeforeunload = function() {
  sendMessage({
    type: 'bye',
    room: room
  });
};

/////////////////////////////////////////////////////////

function createPeerConnection(peerID) {
  try {
    // pc = new RTCPeerConnection(null);
    pc = new RTCPeerConnection(pcConfig);

    pc.onicecandidate = handleIceCandidate;
    pc.onaddstream = handleRemoteStreamAdded;
    pc.onremovestream = handleRemoteStreamRemoved;


    // data채널 생성
    dataChannel = pc.createDataChannel('metaData');
    // data채널 이벤트 콜백 설정
    dataChannel.onopen = () => {
      console.error("Caller DataChannel is open");
      var message = "연결되었습니다.😂😆";
      dataChannel.send(message);
    }
    
    // dataChannel.onmessage = (event) => {
    //   var background = document.getElementById('chat_background');

    //   // 각 메시지 배경 생성
    //   var divChatCh = document.createElement('div');
    //   divChatCh.classList.add('chat');
    //   divChatCh.classList.add('ch1');

    //   // 말풍선에 들어갈 value 생성
    //   var divMessage = document.createElement('div');
    //   divMessage.classList.add('textbox');
    //   divMessage.innerText = event.data;

    //   divChatCh.appendChild(divMessage);
    //   background.appendChild(divChatCh);
    // }

    // data채널 수신측 
    pc.ondatachannel = (event) => {
      const receiveChannel = event.channel;

      receiveChannel.onopen = () => {
        console.error("Callee Data Channel is open");
        var message = "remote datachannel is open";
        receiveChannel.send(message);
      };

      receiveChannel.onmessage = (event) => {
        var background = document.getElementById('chat_background');

        // 각 메시지 배경 생성
        var divChatCh = document.createElement('div');
        divChatCh.classList.add('chat');
        divChatCh.classList.add('ch1');

        // 말풍선에 들어갈 value 생성
        var divMessage = document.createElement('div');
        divMessage.classList.add('textbox');
        divMessage.innerText = event.data;

        divChatCh.appendChild(divMessage);
        background.appendChild(divChatCh);
      }
    }


    // pc.oniceconnectionstatechange = () => {
    //   console.error(pc.iceConncetionState);
    //   if(pc.iceConncetionState === 'connected' || pc.iceConnectionState === 'completed') {
    //     var message = "Hi. I'm " + clientID;
    //     dataChannel.send(message);
    //   }
    // }

    // Peer 객체에 담기.
    addPeer(peerID, pc);

    console.log('Created RTCPeerConnnection');

  } catch (e) {
    console.log('Failed to create PeerConnection, exception: ' + e.message);
    alert('Cannot create RTCPeerConnection object.');
    return;
  }
}

function handleIceCandidate(event) {
  console.log('icecandidate event: ', event);
  if (event.candidate) {
    sendMessage({
      type: 'candidate',
      label: event.candidate.sdpMLineIndex,
      id: event.candidate.sdpMid,
      candidate: event.candidate.candidate
    });
  } else {
    console.log('End of candidates.');
  }
}

function handleCreateOfferError(event) {
  console.log('createOffer() error: ', event);
}

function doCall() {
  console.log('Sending offer to peer');
  // pc.createOffer(setLocalAndSendMessage, handleCreateOfferError);
  pc.createOffer().then(sdp => {
    setLocalAndSendMessage(sdp);
  }).catch(err => {
    handleCreateOfferError(err);
  })

  // 다른 peer가 뒤늦게 들어왔을 때 이 peer가 offer를 날리면 안되니까 초기화
  offerFlag = false;
}

function doAnswer() {
  console.log('Sending answer to peer.');
  // pc.createAnswer().then(
  //   setLocalAndSendMessage,
  //   onCreateSessionDescriptionError
  // );

  pc.createAnswer().then(sdp => {
    setLocalAndSendMessage(sdp);
  }).catch(err => {
    onCreateSessionDescriptionError(err);
  });

  
}

function setLocalAndSendMessage(sessionDescription) {
  // 완료가 된 이후에 sendMessage를 보내도록 수정하니 정상 동작하는 듯 하다가 다시 안함
  pc.setLocalDescription(sessionDescription).then(() => {
    sendMessage(sessionDescription);
  });
  console.log('setLocalAndSendMessage sending message', sessionDescription);
}

function onCreateSessionDescriptionError(error) {
  console.trace('Failed to create session description: ' + error.toString());
}

function handleRemoteStreamAdded(event) {
  console.log('Remote stream added.');

  // const video = document.createElement('video')
  // video.id = 'remoteVideo'+videoCnt;
  // video.srcObject = event.stream;
  // video.autoplay = true
  // video.width = '500'
  // const label = document.createElement('div')
  // document.getElementById('remoteVideos').appendChild(label)
  // document.getElementById('remoteVideos').appendChild(video)

  remoteStream = event.stream;
  remoteVideo.srcObject = remoteStream;
}

function handleRemoteStreamRemoved(event) {
  console.log('Remote stream removed. Event: ', event);
}


function handleRemoteHangup() {
  console.log('Session terminated.');
  stop();
  // connectFlag = false;
}

function stop() {

  peers.forEach((peer, peerId) => {
    peer.data.peerConnection.close();
    peer.data.peerConnection = null;
  })
  // pc.close();
  // pc = null;

  peers.clear();
}

function exitRoom() {
  handleRemoteHangup();
  sendMessage({
    type: 'bye',
    room: room
  });
}

function sendChat() {
  var chatMessage = document.getElementById('chat-message-input');

  var background = document.getElementById('chat_background');

  // 메시지의 배경 생성
  var divChatCh = document.createElement('div');
  divChatCh.classList.add('chat');
  divChatCh.classList.add('ch2');

  // 말풍선에 들어갈 value 생성
  var divMessage = document.createElement('div');
  divMessage.classList.add('textbox');
  divMessage.innerText = chatMessage.value;

  divChatCh.appendChild(divMessage);
  background.appendChild(divChatCh);

  dataChannel.send(chatMessage.value);
  
  // pc.ondatachannel = (event) => {
  //   const receiveChannel = event.channel;
  //   receiveChannel.send(chatMessage.value);
  // }
  chatMessage.value = "";
}

function send_chat(event) {
  if(event.keyCode == 13) {
    sendChat()
  }
}

// main copy_chat.js를 module로 로드하면서 함수가 전역 스코프에 노출안되는 증상 해결
window.joinRoom = joinRoom;
window.sendChat = sendChat;
window.send_chat = send_chat;
window.exitRoom = exitRoom;



/* 1:N 연결용 */
function addPeer(peerId, peerConnection) {
  const peer = new CustomPeer(peerId);
  peer.data = { peerConnection };

  peers.set(peerId, peer);
  console.log(`Peer ${peerId} added.`);
}