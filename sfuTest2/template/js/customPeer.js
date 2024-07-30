// customPeer.js
class CustomPeer {
    constructor(id) {
        this.id = id; // client ID
        this.data = {}; // 추가 데이터를 위한 객체.. (ex RTCPeerConnection)
    }

    addData(key, value) {
        this.data[key] = value;
    }

    getData(key) {
        return this.data[key];
    }
} 

export default CustomPeer;