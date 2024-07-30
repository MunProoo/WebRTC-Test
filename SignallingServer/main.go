package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[string]*Client)
var clientCount = 0

var rooms = make(map[string]*Room)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	var room string

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	connClose := func() {
		fmt.Println("웹소켓 연결 끊김")
		conn.Close()
	}

	defer connClose()
	// defer conn.Close()

	// 웹소켓 연결을 특정하기 위해서 클라이언트로 할당
	clientID := fmt.Sprintf("connection-%d", clientCount)
	client := &Client{
		Id:   clientID,
		Conn: conn,
	}
	clients[clientID] = client

	clientCount++

	for {
		// var msg Message
		// err := conn.ReadJSON(&msg)
		// if err != nil {
		// 	log.Printf("error: %v", err)
		// 	delete(clients, clientID)
		// 	break
		// }

		_, message, err := conn.ReadMessage()
		if err != nil { //  보통 새로고침 시 에러
			log.Printf("error: %v", err)
			delete(clients, clientID)
			if room != "" {
				if len(rooms) > 0 {
					// 방에서 클라이언트 퇴장처리
					delete(rooms[room].Client, clientID)

					// 방에 아무도 없으면 방 삭제처리
					if len(rooms[room].Client) == 0 {
						delete(rooms, room)
					}
				}
			}
			break
		}

		var msg interface{}
		var jsonMsg map[string]interface{} // json인 경우 사용
		if err := json.Unmarshal(message, &jsonMsg); err == nil {
			msg = jsonMsg
		} else {
			msg = strings.Trim(string(message), "\"")
		}

		// var msg map[string]interface{}
		switch v := msg.(type) {
		case map[string]interface{}:
			switch v["type"].(string) {
			case "create or join":
				room = v["room"].(string)
				CreateOrJoinRoom(room, clientID)
			default:
				tmpMap := make(map[string]interface{})
				tmpMap["msg"] = v
				tmpMap["clientID"] = clientID

				if room != "" {
					rooms[room].rommCh <- tmpMap
				}
			}
		case string:
			var receivedMessage = Message{
				Content:  msg.(string),
				ClientID: clientID,
			}
			rooms[room].rommCh <- receivedMessage
		}
		// switch msg["type"] {
		// case "create or join":
		// 	room = msg["room"].(string)
		// 	CreateOrJoinRoom(room, clientID)
		// default:
		// 	rooms[room].rommCh <- msg
		// }
	}
}

func main() {
	http.HandleFunc("/ws", handleConnections)
	// http.Handle("/", http.FileServer(http.Dir("./template/")))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "./template/index_copy.html")
			return
		}
		http.FileServer(http.Dir("./template/")).ServeHTTP(w, r)
	})

	log.Println("http server started on :5000")
	err := http.ListenAndServeTLS(":5000", "public.pem", "private.pem", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func CreateOrJoinRoom(room string, clientID string) {
	fmt.Println("Received request to create or join room : " + room + " by " + clientID)
	client := clients[clientID]

	if _, ok := rooms[room]; !ok {
		fmt.Println("Room is created : " + room + " by " + clientID)
		newRoom := Room{
			Name:   room,
			Client: make(map[string]*Client),
			rommCh: make(chan interface{}),
		}
		// 방에 클라이언트 할당
		newRoom.Client[clientID] = client
		rooms[room] = &newRoom

		message := map[string]interface{}{
			"type": "created",
			"room": room,
		}

		err := client.Conn.WriteJSON(message)
		if err != nil {
			log.Println("WebSocket Conenction Write Err : ", err)
			return
		}

		// room 전용 통신망
		go handleMessage(room)
		return
	}

	if len(rooms[room].Client) > 2 {
		fmt.Println("꽉 찼다")

		message := map[string]interface{}{
			"type": "full",
			"room": room,
		}
		err := client.Conn.WriteJSON(message)
		if err != nil {
			log.Println("WebSocket Conenction Write Err : ", err)
		}
		return
	}

	// Join 코드
	기존룸 := rooms[room]
	기존룸.Client[clientID] = client

	for cID, 기존client := range clients {
		if cID != clientID { // 다른 peer에게 누군가 들어왔다는 알림
			message := map[string]interface{}{
				"type":   "knock",
				"room":   room,
				"client": clientID,
			}

			err := 기존client.Conn.WriteJSON(message)
			if err != nil {
				log.Println("WebSocket Conenction Write Err : ", err)
				return
			}
		} else { // 방에 잘 들어왔다는 알림
			message := map[string]interface{}{
				"type":   "joined",
				"room":   room,
				"client": clientID,
			}

			err := 기존client.Conn.WriteJSON(message)
			if err != nil {
				log.Println("WebSocket Conenction Write Err : ", err)
				return
			}
		}
	}

	fmt.Printf("Client %s connected to room %s \n", clientID, room)
}

// socket 라이브러리의 room 개념 구현
func handleMessage(room string) {
	for {
		msg := <-rooms[room].rommCh
		var message interface{}
		var recvedClientID string
		// var disconnectFlag bool

		switch v := msg.(type) {
		case map[string]interface{}:
			message = v["msg"].(map[string]interface{})
			recvedClientID = v["clientID"].(string)

		case Message:
			mapMessage := map[string]interface{}{
				"type":    "message",
				"room":    room,
				"content": v.Content,
				"client":  v.ClientID,
			}
			message = mapMessage
			recvedClientID = v.ClientID
		}

		for _, client := range rooms[room].Client {
			// 마지막 사람 퇴장 시 방 폭파
			if message.(map[string]interface{})["type"].(string) == "bye" {
				fmt.Println(recvedClientID + " is leaved Room : " + room)
				if len(rooms[room].Client) == 1 {
					fmt.Println(room + " is deleted : ")
					delete(rooms, room)
					return
				} else {
					// 방장이 나갔다면 그 방 재입장이 불가능해지는 버그
					// bye가 전달되면 어차피 방 폭파인데, 상관 없지않나. 근데 got user media가 안되서 연결이 안되는거겠네
				}
			}
			if client.Id == recvedClientID {
				continue
			}

			err := client.Conn.WriteJSON(message)
			if err != nil {
				log.Println("WebSocket Conenction Write Err : ", err)
				return
			}
		}

	}
}
