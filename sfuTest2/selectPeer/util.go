package main

import (
	"encoding/json"
	"log"
)

// 서버가 받고있는 트랙의 리스트
func makeTrackList() []byte {
	var trackList []string
	var terminalMap = make(map[string]struct{})

	for _, val := range trackLocals {
		if _, ok := terminalMap[val.TerminalID]; !ok {
			terminalMap[val.TerminalID] = struct{}{}
			trackList = append(trackList, val.TerminalID)
		}
	}

	message := map[string]interface{}{
		"type":      "trackUpdated",
		"trackList": trackList,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
		return nil
	}
	return data
}
