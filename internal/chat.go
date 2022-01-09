package internal

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type Room struct {
	name        string
	clients     map[int]*Client
	clientNames map[int]string
}

func NewRoom(name string) *Room {
	r := &Room{name: name}
	r.clients = make(map[int]*Client)
	r.clientNames = make(map[int]string)
	return r
}

func (r *Room) getClientNames() []string {
	clientNames := make([]string, len(r.clients))
	index := 0
	for _, name := range r.clientNames {
		clientNames[index] = name
		index += 1
	}
	return clientNames
}

type ClientMessage struct {
	clientId   int
	rawMessage []byte
}

type Client struct {
	id   int
	conn *websocket.Conn
}

var idClientGenerator = Generator{}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{id: idClientGenerator.generateId(), conn: conn}
}

type ChatServer struct {
	Address   string
	Pattern   string
	clients   map[int]*Client
	onConnect chan *websocket.Conn
	onClose   chan *ClientMessage
	onMessage chan *ClientMessage
	chatRooms map[string]*Room
	upgrader  *websocket.Upgrader
}

func (cs *ChatServer) Run() {
	cs.clients = make(map[int]*Client)
	cs.chatRooms = make(map[string]*Room)
	cs.onConnect = make(chan *websocket.Conn)
	cs.onClose = make(chan *ClientMessage)
	cs.onMessage = make(chan *ClientMessage)
	cs.upgrader = &websocket.Upgrader{}
	cs.upgrader.CheckOrigin = func(request *http.Request) bool { return true } // TODO: implement check origin function

	go func() {
		http.HandleFunc(cs.Pattern, cs.connectionRequestHandler)
		if err := http.ListenAndServe(cs.Address, nil); err != http.ErrServerClosed {
			log.Fatalf("Could not start web socket server: %s\n", err)
		}
	}()
	log.Printf("Chat server is listening on %s.\n", cs.Address)
	for {
		select {
		case conn := <-cs.onConnect:
			c := NewClient(conn)
			cs.clients[c.id] = c
			log.Printf("New connection established: %d.\n", c.id)
			go cs.readFromClient(c)
		case clientMsg := <-cs.onClose:
			cs.closeClient(clientMsg.clientId)
			log.Printf("Connection %d closed.\n", clientMsg.clientId)
			// TODO: remove clients from rooms
		case clientMsg := <-cs.onMessage:
			log.Printf("New message received from client %d.\n", clientMsg.clientId)
			msg, err := ParseClientMessages(clientMsg.rawMessage)
			client := cs.clients[clientMsg.clientId]
			if err != nil {
				cs.writeToClient(client, NewUnableToParseMessage())
				log.Printf("Unable to parse client message %s.\n", clientMsg.rawMessage)
			}
			switch m := msg.(type) {
			case Text:
				//check if client is in room
				//broadcast message to other clients
			case JoinRoom:
				roomName := m.RoomName
				userName := m.UserName
				var chatRoom *Room
				if cs.chatRooms[roomName] == nil {
					chatRoom = NewRoom(roomName)
					cs.chatRooms[chatRoom.name] = chatRoom
					log.Printf("New room %s has been created.\n", roomName)
				}
				chatRoom = cs.chatRooms[roomName]
				chatRoom.clients[client.id] = client
				chatRoom.clientNames[client.id] = userName
				cs.writeToClient(client, NewSuccessJoinRoomMessage(roomName))
				log.Printf("Client %d joined room %s with name %s.\n", client.id, roomName, userName)
			}
		}
	}
}

func (cs *ChatServer) connectionRequestHandler(responseWriter http.ResponseWriter, request *http.Request) {
	conn, err := cs.upgrader.Upgrade(responseWriter, request, nil)
	if err != nil {
		log.Println(err)
		return
	}
	cs.onConnect <- conn
}

func (cs *ChatServer) readFromClient(c *Client) {
	for {
		_, p, err := c.conn.ReadMessage()
		msg := &ClientMessage{clientId: c.id, rawMessage: p}
		if err != nil {
			log.Println(err)
			cs.onClose <- msg
			return
		}
		cs.onMessage <- msg
	}
}

func (cs *ChatServer) writeToClient(c *Client, rawMessage []byte) {
	err := c.conn.WriteMessage(websocket.TextMessage, rawMessage)
	if err != nil {
		log.Printf("Unable to send message %s to client %d.\n", string(rawMessage), c.id)
	}
}

func (cs *ChatServer) closeClient(id int) {
	err := cs.clients[id].conn.Close()
	if err != nil {
		log.Println(err)
	}
	delete(cs.clients, id)
}
