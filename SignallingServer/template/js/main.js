'use strict';

var isChannelReady = false;
var isInitiator = false;
var isStarted = false;
var localStream;
var pc;
var remoteStream;
var turnReady;

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

socket.onopen = function() {
  console.log("websocket connection opened");
}

function joinRoom() {
  room = document.getElementById('roomName').value;

  console.log(JSON.stringify({type:'create or join', room:room}));
  socket.send(JSON.stringify({type:'create or join', room:room}));

  navigator.mediaDevices.getUserMedia({
    audio: false,
    video: true
  })
  .then(gotStream)
  .catch(function(e) {
    console.log('getUserMedia() error: ' + e.name);
  });
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
      isInitiator = true;
      break;
    case 'full':
      console.log('Room ' + message.room + ' is full');
      break;
    case 'join':
      console.log('This peer is the initiator of room ' + room + '!');
      isChannelReady = true;
      break;
    case 'joined':
      console.log('Another peer made a request to join room ' + room);
      isInitiator = true;
      isChannelReady = true;
      break;
    case 'log':
      console.log.apply(console, array);
      break;
    case 'offer': // PeerA -> PeerB 즉 PeerB에서 처리됨
      console.log("offer");
      if(!isInitiator && !isStarted) {
        maybeStart();
      }
      pc.setRemoteDescription(new RTCSessionDescription(message));
      doAnswer();
      break;
    case 'answer': // PeerB -> PeerA 즉 PeerA에서 처리됨
      console.log("answer");
      if (isStarted){
        pc.setRemoteDescription(new RTCSessionDescription(message));
      }
      break;
    case 'candidate':
      if (isStarted){
        var candidate = new RTCIceCandidate({
          sdpMLineIndex : message.label,
          candidate : message.candidate
        });
        pc.addIceCandidate(candidate);
      }
      break;
    case 'bye':
      if(isStarted){
        handleRemoteHangup();
      }
  }

}

function handleMessage(message) {
  console.log('Client received message from : ' + message.client);

  var content = message.content
  if(content == 'got user media') {
    maybeStart()
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
  sendMessage('got user media');
  if (isInitiator) {
    maybeStart();
  }
}

var constraints = {
  video: true
};

console.log('Getting user media with constraints', constraints);

// if (location.hostname !== 'localhost') {
//   requestTurn(
//     'https://computeengineondemand.appspot.com/turn?username=41784574&key=4080218913'
//   );
// }

function maybeStart() {
  console.log('>>>>>>> maybeStart() ', isStarted, localStream, isChannelReady);
  if (!isStarted && typeof localStream !== 'undefined' && isChannelReady) {
    console.log('>>>>>> creating peer connection');
    createPeerConnection();
    pc.addStream(localStream);
    isStarted = true;
    console.log('isInitiator', isInitiator);
    if (isInitiator) {
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

function createPeerConnection() {
  try {
    // pc = new RTCPeerConnection(null);
    pc = new RTCPeerConnection(pcConfig);
    pc.onicecandidate = handleIceCandidate;
    pc.onaddstream = handleRemoteStreamAdded;
    pc.onremovestream = handleRemoteStreamRemoved;
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
  pc.createOffer(setLocalAndSendMessage, handleCreateOfferError);
}

function doAnswer() {
  console.log('Sending answer to peer.');
  pc.createAnswer().then(
    setLocalAndSendMessage,
    onCreateSessionDescriptionError
  );
}

function setLocalAndSendMessage(sessionDescription) {
  pc.setLocalDescription(sessionDescription);
  console.log('setLocalAndSendMessage sending message', sessionDescription);
  sendMessage(sessionDescription);
}

function onCreateSessionDescriptionError(error) {
  console.trace('Failed to create session description: ' + error.toString());
}

function requestTurn(turnURL) {
  var turnExists = false;
  for (var i in pcConfig.iceServers) {
    if (pcConfig.iceServers[i].urls.substr(0, 5) === 'turn:') {
      turnExists = true;
      turnReady = true;
      break;
    }
  }
  if (!turnExists) {
    console.log('Getting TURN server from ', turnURL);
    // No TURN server. Get one from computeengineondemand.appspot.com:
    var xhr = new XMLHttpRequest();
    xhr.onreadystatechange = function() {
      if (xhr.readyState === 4 && xhr.status === 200) {
        var turnServer = JSON.parse(xhr.responseText);
        console.log('Got TURN server: ', turnServer);
        pcConfig.iceServers.push({
          'urls': 'turn:' + turnServer.username + '@' + turnServer.turn,
          'credential': turnServer.password
        });
        turnReady = true;
      }
    };
    xhr.open('GET', turnURL, true);
    xhr.send();
  }
}

function handleRemoteStreamAdded(event) {
  console.log('Remote stream added.');
  remoteStream = event.stream;
  remoteVideo.srcObject = remoteStream;
}

function handleRemoteStreamRemoved(event) {
  console.log('Remote stream removed. Event: ', event);
}


function handleRemoteHangup() {
  console.log('Session terminated.');
  stop();
  isInitiator = false;
}

function stop() {
  isStarted = false;
  pc.close();
  pc = null;
}

function exitRoom() {
  handleRemoteHangup();
  sendMessage({
    type: 'bye',
    room: room
  });
}
