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
}

var messageHub = hub{
	register:      make(chan *connection),
	unregister:    make(chan *connection),
	addSubscriber: make(chan *Subscriber),
	connections:   make(map[*connection]bool),
}

func (h *hub) run() {
	if *Debug {
		log.Println("hub start")
	}

	for {
		select {
		case c := <-h.register:
			h.connections[c] = true

		case c := <-h.unregister:
			if *Debug {
				log.Println("h.unregister fired")
			}
			delete(h.connections, c)
			close(c.send)

		case sub := <-h.addSubscriber:
			if *Debug {
				log.Println("h.addSubscriber fired")
			}
			conn := sub.Conn
			conn.nc.Subscribe(sub.Topic, func(msg *nats.Msg) {
				if *Debug {
					log.Println("Connection", conn, "got message: ", string(msg.Data))
				}

				conn.send <- msg.Data
			})
		}
	}

	panic("hub end of life")
}
