'use strict';

var pcConfig = {
    'iceServers': [{
      // 'urls': 'stun:stun.l.google.com:19302',
    'urls': [
     'turn:192.168.30.186:8888?transport=udp', 
     'turn:192.168.30.186:8888?transport=tcp'],
    // 'urls': [
    //  'turn:211.207.68.244:8888?transport=udp', 
    //  'turn:211.207.68.244:8888?transport=tcp'],
    'username':'foo',
    'credential' :'bar'
    }]
};

let pc;
let terminalID;
let trackMap = new Map();
let receiveChannel;

let webCamStream;
let displayStream;
var address = window.location.host;
var ws;
var receiverID; // 전화받은 사람

function createConnection() {
  terminalID = document.getElementById('my_terminal_id').value;
  if(terminalID == "") {
    alert("단말기 아이디를 입력하세요.");
    return;
  } 
 
  var videoConstraints = {
    video:{
      width: {ideal:768}, // Face Premium 해상도
      height: {ideal:480},
      frameRate:{ideal:30}
    },
    audio: false
  };
    
  // navigator.mediaDevices.getDisplayMedia({ video: true, audio: false })
  const getWebCamStream = navigator.mediaDevices.getUserMedia(videoConstraints).then(stream => {
    webCamStream = stream;
  });

  // const getDisplayStream =navigator.mediaDevices.getDisplayMedia({ video: true, audio: false }).then(stream => {
  //   displayStream = stream;
  // });

  // 두 스트림을 다 받고난 후에 실행
  // Promise.all([getWebCamStream, getDisplayStream]).then(() => {
  Promise.all([getWebCamStream]).then(() => {
    if (ws) {
      console.log("이미 접속해있음");
      return;
    }

    pc = new RTCPeerConnection(pcConfig);
    pc.ontrack = handleOnTrack;
    handleDataChannel();

    document.getElementById('localVideo').srcObject = webCamStream;
    // document.getElementById('localDisplay').srcObject = displayStream;

    webCamStream.getTracks().forEach(track => pc.addTrack(track, webCamStream))
    // displayStream.getTracks().forEach(track => pc.addTrack(track, displayStream))

    var address = window.location.host;
    ws = new WebSocket('wss://'+address+'/ws');

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

    ws.onopen = function(evt) {
      ws.send(JSON.stringify({event: 'init', 'terminalID': terminalID}))
    }

    ws.onmessage = function(evt) {
      let msg = JSON.parse(evt.data);
      handleWebsocketMessage(msg);
    }
    ws.onerror = function(evt) {
      console.log("ERROR: " + evt.data)
    }
  }).catch(err => window.alert(err));
}



function handleOnTrack(event) {
  // if (event.track.kind === 'audio' || event.track.kind === 'video') {
  //     return
  // }

  console.log("onTrack");
  // Video, Audio 트랙 하나로 합쳐서 송출하도록
  // Video, Audio 싱크 문제 있을 수 있음...
  // var stream = trackMap.get(event.streams[0].id);
  // stream.addTrack(event.track);

  var stream = event.streams[0];
  var el = document.getElementById('video-'+event.streams[0].id);
  // var label = document.getElementById(event.streams[0].id);
  el.srcObject = stream;

  // Mute 상태가 되었을 경우 중지가 되는 경우 있으므로 재생시도.
  event.track.onmute = function(event) {
    el.play()
  }

  event.streams[0].onremovetrack = ({track}) => {
      if (el.parentNode && track.kind === 'video') {
        el.parentNode.parentNode.removeChild(el.parentNode)
        // el.parentNode.removeChild(el);
        // label.parentNode.removeChild(label);
      }
  }
}

function handleDataChannel() {
  // data채널 수신측 
  pc.ondatachannel = (event) => {
    receiveChannel = event.channel;

    receiveChannel.onopen = () => {
      console.error("Data Channel is open");
      
      // var message = userName + "님이 입장하였습니다.😂😆";
      // const data = JSON.stringify({message: message, userName:userName, type:"chat"});

      // 나의 terminalID 전달 -> 여기 없애야함
      const data = JSON.stringify({terminalID:terminalID, type:"init"});
      receiveChannel.send(data);
    };

    receiveChannel.onmessage = (event) => {
      if(typeof event.data === 'string') { // 서버와 연결 완료 메시지
        showChattingMessage(event);
      } else { // TextDecoding 해야하는 메시지들
        const decoder = new TextDecoder('utf-8');
        var msg = JSON.parse(decoder.decode(event.data));

        switch(msg.type) {
          case "trackUpdated":
            console.log("가능한 트랙리스트 받는 중");
            console.log(msg);
            if(msg.trackList != null) {
              appendTerminalIDs(msg.trackList);
            }

            break
          case "metadata":
            console.log("metadata (출처) 수신");
            createVideo(msg);
            break;
          case "chat":
            console.log("chatting 수신");
            showChattingMessage(event);
            break;
          case "peerList":
            console.log("peerList 수신");
            appendPeerList(msg.peerList);
            break;
          case "callOffer":
            console.log("통화요청 수신");
            callAnswer(msg);
            break;
          case "callAnswer":
            console.log("통화요청 결과 수신");
            callResult(msg);
            break;
          case "hangUp":
            hangUp("complete");
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
    
      const data = JSON.stringify({message: chatMessage.value , terminalID:terminalID, type:"chat"});
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
    divProfile.innerText = msg.terminalID;
    // divProfile.style.backgroundColor = "white";
    divMessage.innerText = msg.message;
  }

  divChatCh.appendChild(divMessage);
  background.appendChild(divChatCh);
}

function createVideo(msg) {
  // console.log(msg.streamID);
  if(msg.kind === 'audio') {
    return;
  }

  var newTrack = new MediaStream();
  trackMap.set(msg.streamID, newTrack);

  let bg = document.createElement('div');
  let label = document.createElement('div');
  label.id = msg.streamID;
  label.innerText = msg.terminalID;
  label.classList.add('video-label');

  let el = document.createElement(msg.kind);
  el.id = 'video-'+msg.streamID;
  el.autoplay = true;
  el.controls = true;
  el.playsInline = true;
  // el.width = 160;
  el.width = 300;
  // el.height = 120;
  el.height = 250;

  bg.id = 'bg-'+msg.terminalID;
  bg.appendChild(label);
  bg.appendChild(el)
  document.getElementById('remoteVideos').appendChild(bg);

}

function appendTerminalIDs(trackList) {
  var el = document.getElementById('terminal_ids');
  el.options.length = 0; // 기존 옵션 전부 삭제

  var option = document.createElement('option');
  option.value = "";
  option.innerText = "==선택없음==";
  el.appendChild(option);

  trackList.forEach(terminalID => {
    option = document.createElement('option');
    option.value = terminalID;
    option.innerText = terminalID;
    el.appendChild(option);
  });
}

function selectTerminal(e) {
  // var el = document.getElementById('terminal_ids');
  var selectedValue = [];

  for(let i=0; i < e.options.length; i++) {
    const option = e.options[i];
    if(option.selected) {
      selectedValue.push(option.value);
    }
  }
  console.log('선택한 단말기 : ' + selectedValue);
  receiveChannel.send(JSON.stringify({array:selectedValue, type:"trackOffer"}));
}

function handleWebsocketMessage(msg) {
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

function NoCamConnection() {
  terminalID = document.getElementById('my_terminal_id').value;
  if(terminalID == "") {
    alert("단말기 아이디를 입력하세요.");
    return;
  } 
  
    if (ws) {
      console.log("이미 연결됨");
      return;
    }

    pc = new RTCPeerConnection(pcConfig);
    pc.ontrack = handleOnTrack;
    handleDataChannel();

    var address = window.location.host;
    ws = new WebSocket('wss://'+address+'/ws');

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

    ws.onopen = function(evt) {
      ws.send(JSON.stringify({event: 'init', 'terminalID': terminalID}))
    }

    ws.onmessage = function(evt) {
      let msg = JSON.parse(evt.data);
      handleWebsocketMessage(msg);
    }
    ws.onerror = function(evt) {
      console.log("ERROR: " + evt.data)
    }
}

function appendPeerList(peerList) {
  var el = document.getElementById('peerList');
  el.options.length = 0; // 기존 옵션 전부 삭제

  var option = document.createElement('option');
  option.value = "";
  option.innerText = "==선택없음==";
  el.appendChild(option);


  for (const peer of peerList) {
    if(peer == terminalID) {
      continue
    }
    option = document.createElement('option');
    option.value = peer;
    option.innerText = peer;
    el.appendChild(option);
  }

  // peerList.forEach(peer => {}); // forEach는 continue를 쓸 수 없다
}

// ------------------------------------통화요청 관련 -----------------------------------------------------------------------
function selectPeer(e) {
  console.log(e.value);
  if(e.value == "") {
    hangUp("");
    return;
  }

  var message = e.value + "에게 통화 요청하시겠습니까?";
  if(confirm(message)) {
    receiveChannel.send(JSON.stringify({type:"callOffer",receiverID:e.value,TerminalID:terminalID}));
  } else {
    document.getElementById('peerList').value = "";
  }
  
}

function callAnswer(msg) {
  var message = msg.callerID + " (으)로부터 통화요청이 들어왔습니다. 승인하시겠습니까?";
  if(confirm(message)) {
    alert("연결을 시작합니다.");
    document.getElementById('peerList').value = msg.callerID;
    // 통화 연결 승인 전달
    receiveChannel.send(JSON.stringify({type:"callAnswer",Message:"true", TerminalID:terminalID, CallerID:msg.callerID}));
    // caller 트랙 요청
    receiveChannel.send(JSON.stringify({type:"callComplete", TerminalID:terminalID, ReceiverID:msg.callerID}));
    receiverID = msg.callerID;
    
  } else {
    receiveChannel.send(JSON.stringify({type:"callAnswer",Message:"false",TerminalID:terminalID, CallerID:msg.callerID}));
  }
}

function callResult(msg) {
  if (msg.answer) {
    receiverID = msg.receiverID;
    receiveChannel.send(JSON.stringify({type:"callComplete",TerminalID:terminalID, ReceiverID:msg.receiverID}));
  } else {
    var message = msg.receiverID + "가 통화를 거부하였습니다.";
    alert(message);

    document.getElementById('peerList').value = "";
  }
}

function hangUp(message) {
  // 통화 종료 로직
  receiveChannel.send(JSON.stringify({type:"hangUp", Message:message,TerminalID:terminalID, ReceiverID:receiverID}));
  document.getElementById('peerList').value = "";
  alert("통화가 종료되었습니다.");
}