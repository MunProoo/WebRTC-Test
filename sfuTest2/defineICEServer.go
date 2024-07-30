package main

import (
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type ICEServerProcessor struct {
	Rooms           map[string]*Room
	Clients         map[string]*Client // 전체 client
	PeerConnections map[string]*webrtc.PeerConnection
	Api             *webrtc.API
	OutputTracks    map[string]*webrtc.TrackLocalStaticRTP // Peer의 Media Stream들
}

type Client struct {
	Id   string
	Conn *websocket.Conn
}

type Room struct {
	Name   string
	Client map[string]*Client // room에 있는 client
	rommCh chan interface{}
}

type Message struct {
	Content  string
	ClientID string
}
