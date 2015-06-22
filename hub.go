package main

import (
	"log"

	"github.com/apcera/nats"
)

type hub struct {
	// Registered connections.
	connections map[*connection]bool

	// register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection

	// Add topic subscription for connection
	addSubscriber chan *Subscriber

	// Registered Nsq Readers
	//nsqReaders map[string]*NsqTopicReader // NSQ topic -> nsqReader
}

var messageHub = hub{
	register:      make(chan *connection),
	unregister:    make(chan *connection),
	addSubscriber: make(chan *Subscriber),
	connections:   make(map[*connection]bool),
	//nsqReaders:    make(map[string]*NsqTopicReader),
}

func (h *hub) run() {
	log.Printf("hub start of life\n")
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true

		case c := <-h.unregister:
			log.Println("h.unregister fired")
			delete(h.connections, c)
			log.Println("connection deleted from hub's poll OK")
			close(c.send)

		case sub := <-h.addSubscriber:
			log.Println("h.addSubscriber fired")
			conn := sub.Conn
			conn.nc.Subscribe(sub.Topic, func(msg *nats.Msg) {
				log.Println(conn, "got message: ", string(msg.Data))

				conn.send <- msg.Data
			})
		}
	}

	panic("hub end of life")
}
