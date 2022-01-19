package internal

import (
	"encoding/json"
)

type Envelope struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Messages sent by client
const (
	JoinRoomType  = "JOIN_ROOM"
	LeaveRoomType = "LEAVE_ROOM"
	TextType      = "TEXT"
)

type JoinRoom struct {
	RoomName   string `json:"roomName"`
	ClientName string `json:"clientName"`
}

type LeaveRoom struct {
	RoomName string `json:"roomName"`
}

type Text struct {
	Text string `json:"text"`
}

func ParseClientMessages(rawMessage []byte) (interface{}, error) {
	var msg json.RawMessage
	env := Envelope{Data: &msg}
	err := json.Unmarshal(rawMessage, &env)
	if err != nil {
		return err, nil
	}

	var parsedMsg interface{}
	switch env.Type {
	case JoinRoomType:
		var joinRoomData JoinRoom
		err := json.Unmarshal(msg, &joinRoomData)
		if err != nil {
			return nil, err
		}
		parsedMsg = joinRoomData
	case LeaveRoomType:
		var leaveRoomData LeaveRoom
		err := json.Unmarshal(msg, &leaveRoomData)
		if err != nil {
			return nil, err
		}
		parsedMsg = leaveRoomData
	case TextType:
		var textData Text
		err := json.Unmarshal(msg, &textData)
		if err != nil {
			return nil, err
		}
		parsedMsg = textData
	}
	return parsedMsg, nil
}

// Messages sent by server
const (
	SuccessJoinRoomType  string = "SUCCESS_JOIN_ROOM"
	SuccessLeaveRoomType        = "SUCCESS_LEAVE_ROOM"
	ClientListType              = "CLIENT_LIST"
)

type SuccessJoinRoom struct {
	RoomName string `json:"roomName"`
}

type GetAllClientNames struct {
	ClientNames []string `json:"clientNames"`
}

func NewSuccessJoinRoomMessage(roomName string) []byte {
	env := &Envelope{Type: SuccessJoinRoomType}
	env.Data = &SuccessJoinRoom{roomName}
	jsonMsg, _ := json.Marshal(env)
	return jsonMsg
}

func NewSuccessLeaveRoomMessage(roomName string) []byte {
	env := &Envelope{Type: SuccessLeaveRoomType}
	env.Data = &SuccessJoinRoom{roomName}
	jsonMsg, _ := json.Marshal(env)
	return jsonMsg
}

func NewClientNamesMessage(clientNames []string) []byte {
	env := &Envelope{Type: ClientListType}
	env.Data = &GetAllClientNames{clientNames}
	jsonMsg, _ := json.Marshal(env)
	return jsonMsg
}
