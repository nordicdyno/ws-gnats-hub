package main

import (
	"encoding/json"
	"log"
	//"net/http"
	"time"
	//"io/ioutil"

	"github.com/apcera/nats"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

type connection struct {
	// The websocket connection.
	ws *websocket.Conn
	// Buffered channel of outbound messages.
	send chan []byte
	// gnats connection
	nc *nats.Conn
}

type Subscriber struct {
	Conn  *connection
	Topic string
}

type SubMessage struct {
	Event   string
	Channel string `json:"channel"`
	// Data    SubMessageData `json:"data"` // just ignore it
}

// write writes a message with the given message type and payload.
func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func (c *connection) reader() {
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}
		c.processMessage(message)
	}
	c.ws.Close()
}

// process subscription
func (c *connection) processMessage(msg []byte) {
	message := SubMessage{}
	err := json.Unmarshal(msg, &message)
	if err != nil {
		log.Println("ERROR: invalid JSON subscribe data" + string(msg))
		return
	}
	log.Printf("message+: %v\n", message)

	channel := DefaultChat + "." + message.Channel
	s := Subscriber{Conn: c, Topic: channel}
	log.Printf("send to channel: %v\n", s)
	messageHub.addSubscriber <- &s
}

// TODO: add error channel ?
func (c *connection) writer() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			log.Println("send Ping")
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}

}
