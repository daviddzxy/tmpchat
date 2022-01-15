package internal

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type Room struct {
	name    string
	clients map[int]*Client
}

func NewRoom(name string) *Room {
	r := &Room{name: name}
	r.clients = make(map[int]*Client)
	return r
}

func (r *Room) getClientNames() []string {
	clientNames := make([]string, len(r.clients))
	index := 0
	for _, c := range r.clients {
		clientNames[index] = c.clientNameByRoomName[r.name]
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
	// Maps room id to clients name in give room
	clientNameByRoomName map[string]string
}

var idClientGenerator = Generator{}

func NewClient(conn *websocket.Conn) *Client {
	c := &Client{id: idClientGenerator.generateId(), conn: conn}
	c.clientNameByRoomName = make(map[string]string)
	return c
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

func NewChatServer(address string, pattern string) *ChatServer {
	cs := &ChatServer{Address: address, Pattern: pattern}
	cs.clients = make(map[int]*Client)
	cs.chatRooms = make(map[string]*Room)
	cs.onConnect = make(chan *websocket.Conn)
	cs.onClose = make(chan *ClientMessage)
	cs.onMessage = make(chan *ClientMessage)
	cs.upgrader = &websocket.Upgrader{}
	cs.upgrader.CheckOrigin = func(request *http.Request) bool { return true } // TODO: implement check origin function
	return cs
}

func (cs *ChatServer) Run() {
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
			// TODO: remove clients from rooms
			// Remove client from rooms
			// Broadcast that client left
			cs.closeClient(clientMsg.clientId)
			log.Printf("Connection %d closed.\n", clientMsg.clientId)
		case clientMsg := <-cs.onMessage:
			log.Printf("New message %s received from client %d.\n", string(clientMsg.rawMessage), clientMsg.clientId)
			msg, err := ParseClientMessages(clientMsg.rawMessage)
			if err != nil {
				log.Printf("Unable to parse client message %s.\n", clientMsg.rawMessage)
			}
			client := cs.clients[clientMsg.clientId]
			switch m := msg.(type) {
			case Text:
				//check if client is in room
				//broadcast message to other clients
			case JoinRoom:
				cs.handleJoinRoomMessage(m, client)
			}
		}
	}
}

func (cs *ChatServer) handleJoinRoomMessage(m JoinRoom, c *Client) {
	var r *Room
	if cs.chatRooms[m.RoomName] == nil {
		r = NewRoom(m.RoomName)
		cs.chatRooms[r.name] = r
		log.Printf("New room %s has been created.\n", m.RoomName)
	}
	r = cs.chatRooms[m.RoomName]
	if r.clients[c.id] == nil {
		r.clients[c.id] = c
		c.clientNameByRoomName[r.name] = m.ClientName
		cs.writeToClient(c, NewSuccessJoinRoomMessage(m.RoomName))
		cs.writeToClient(c, NewClientNamesMessage(m.RoomName, r.getClientNames()))
		log.Printf("Client %d joined room %s with name %s.\n", c.id, m.RoomName, m.ClientName)
	}
}

func removeClientFromAllRooms(id int) {

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
	log.Printf("Message %s sent to client %d\n", string(rawMessage), c.id)
	// TODO: return and handle error if write failed
}

func (cs *ChatServer) closeClient(id int) {
	err := cs.clients[id].conn.Close()
	if err != nil {
		log.Println(err)
	}
	delete(cs.clients, id)
}
