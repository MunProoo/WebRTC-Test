package main

import "github.com/gorilla/websocket"

type Client struct {
	Id   string
	Conn *websocket.Conn
}

type Room struct {
	Name   string
	Client map[string]*Client
	rommCh chan interface{}
}

type Message struct {
	Content  string
	ClientID string
}
