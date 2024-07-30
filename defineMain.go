package main

import "github.com/pion/webrtc/v4"

type PeerConnectionManager struct {
	PeerConnection *webrtc.PeerConnection
	OutputTracks   map[string]*webrtc.TrackLocalStaticRTP
}
