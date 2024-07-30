'use strict';

var pcConfig = {
    'iceServers': [{
    'urls': 'turn:192.168.30.186:3478',
    'username':'foo',
    'credential' :'bar'
    // 'urls': 'stun:stun.l.google.com:19302'
    }]
};

let pc;
let userName;
let trackMap = new Map();

function createConnection() {
  userName = document.getElementById('user_name').value;
  if(userName == "") {
    alert("사용자 이름를 입력하세요.");
    return;
  }

  navigator.mediaDevices.getUserMedia({ video: true, audio: false })
  .then(stream => {
    pc = new RTCPeerConnection(pcConfig);
    pc.ontrack = handleOnTrack;

    handleDataChannel();

    document.getElementById('localVideo').srcObject = stream

    stream.getTracks().forEach(track => pc.addTrack(track, stream))

    var address = window.location.host;
    var ws = new WebSocket('wss://'+address+'/ws');
    pc.onicecandidate = e => {
    if (!e.candidate) {
        return
    }

    ws.send(JSON.stringify({event: 'candidate', data: JSON.stringify(e.candidate)}))
    }

    ws.onclose = function(evt) {
      window.alert("Websocket has closed");
      window.location.reload();
    }

    ws.onmessage = function(evt) {
      let msg = JSON.parse(evt.data)
      if (!msg) {
          return console.log('failed to parse msg')
      }

      switch (msg.event) {
        case 'offer':
        let offer = JSON.parse(msg.data)
        if (!offer) {
            return console.log('failed to parse answer')
        }
        pc.setRemoteDescription(offer)
        pc.createAnswer().then(answer => {
            pc.setLocalDescription(answer)
            ws.send(JSON.stringify({event: 'answer', data: JSON.stringify(answer)}))
        })
        return

        case 'candidate':
        let candidate = JSON.parse(msg.data)
        if (!candidate) {
            return console.log('failed to parse candidate')
        }

        pc.addIceCandidate(candidate)
      }
    }

    ws.onerror = function(evt) {
    console.log("ERROR: " + evt.data)
    }
  }).catch(window.alert)
}



function handleOnTrack(event) {
  if (event.track.kind === 'audio') {
      return
  }

  console.log("onTrack");
  var stream = event.streams[0]
  var el = document.getElementById('video-'+stream.id);
  var label = document.getElementById(stream.id);
  el.srcObject = stream;

  // let el = document.createElement(event.track.kind);
  // el.srcObject = event.streams[0];
  // el.autoplay = true;
  // el.controls = true;
  // el.width = 160;
  // el.height = 120;
  // document.getElementById('remoteVideos').appendChild(el)

  // let label = document.createElement('div');
  // label.id = event.streams[0].id;
  // document.getElementById('remoteVideos').appendChild(label);

  event.track.onmute = function(event) {
    el.play()
  }

  event.streams[0].onremovetrack = ({track}) => {
      if (el.parentNode) {
        el.parentNode.removeChild(el);
        label.parentNode.removeChild(label);
      }
  }
}

function handleDataChannel() {
  // data채널 수신측 
  pc.ondatachannel = (event) => {
    const receiveChannel = event.channel;

    receiveChannel.onopen = () => {
      console.error("Data Channel is open");
      
      var message = userName + "님이 입장하였습니다.😂😆";
      const data = JSON.stringify({message: message, userName:userName, type:"chat"});
      receiveChannel.send(data);
    };

    receiveChannel.onmessage = (event) => {
      if(typeof event.data === 'string') { // 서버와 연결 완료 메시지
        showChattingMessage(event);
      } else { // TextDecoding 해야하는 메시지들
        const decoder = new TextDecoder('utf-8');
        var msg = JSON.parse(decoder.decode(event.data));

        console.log(msg);
        switch(msg.type) {
          case "metadata":
            console.log("metadata (출처)수신");
            createVideo(msg);
            
            break;
          case "chat":
            console.log("chatting 수신");
            showChattingMessage(event);
            break;
        }
      }


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
    
      const data = JSON.stringify({message: chatMessage.value , userName:userName, type:"chat"});
      receiveChannel.send(data);
      chatMessage.value = "";
    }

    function send_chat(event) {
      if(event.keyCode == 13) {
        sendChat()
      }
    }

    // 버튼 클릭 이벤트 핸들러
    document.getElementById('chat-message-submit').addEventListener('click',sendChat)

    // 인풋창 엔터 이벤트 핸들러
    document.getElementById('chat-message-input').addEventListener('keypress', send_chat);

    
  }
}

function showChattingMessage(event) {
  var background = document.getElementById('chat_background');

  // 각 메시지 배경 생성
  var divChatCh = document.createElement('div');
  divChatCh.classList.add('chat');
  divChatCh.classList.add('ch1');

  // 이름 추가 하고픔
  var divProfile = document.createElement('div');

  var divIcon = document.createElement('i');

  divProfile.appendChild(divIcon);
  divChatCh.appendChild(divProfile);


  // 말풍선에 들어갈 value 생성
  var divMessage = document.createElement('div');
  divMessage.classList.add('textbox');

  if(typeof event.data === 'string') { // 서버와 연결 완료 메시지
    divMessage.innerText = event.data;
  } else { // 채팅 메시지
    const decoder = new TextDecoder('utf-8');
  
    var msg = JSON.parse(decoder.decode(event.data));
    divProfile.innerText = msg.userName;
    // divProfile.style.backgroundColor = "white";
    divMessage.innerText = msg.message;
  }

  divChatCh.appendChild(divMessage);
  background.appendChild(divChatCh);
}

function createVideo(msg) {
  let label = document.createElement('div');
  label.id = msg.streamID;
  label.innerText = msg.terminalID;
  label.classList.add('video-label');

  let el = document.createElement(msg.kind);
  el.id = 'video-'+msg.streamID;
  el.autoplay = true;
  el.controls = true;
  el.width = 160;
  el.height = 120;

  document.getElementById('remoteVideos').appendChild(label);
  document.getElementById('remoteVideos').appendChild(el)
}