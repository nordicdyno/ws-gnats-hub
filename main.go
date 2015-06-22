package main

// TODO :   add statistic on /stat url
//          cleanup code (grep TODO/FIXME/XXX)
//          add flags
//          Add tests
// MAYBE: change topic/channels naming schema (add prefix, or allow real names)

import (
	//"bytes"
	//"encoding/json"
	"flag"
	"io/ioutil"
	//"fmt"
	//"go/build"

	"log"
	"net/http"
	//"os"
	//"path/filepath"
	//"strconv"
	"time"

	"github.com/gin-gonic/gin"
	//"github.com/guregu/kami"

	"github.com/apcera/nats"
	"github.com/gorilla/websocket"
	//"github.com/guregu/kami"
)

var (
	addr  = flag.String("addr", ":8000", "http service address")
	serve = flag.String("serve", "", "serve dir with templates")
	Debug = flag.Bool("debug", false, "print a lot!")
)

// default prefix
const (
	DefaultChat = "match"
)

type PostMessage struct {
	Channel string `json:"channel"`
	//Data    json.RawMessage  `json:"data"`
	Data map[string]interface{}
}

/*
type PostMessageData struct {
	Content        string
	Timestamp      uint64
	Type           string
	UserEmail      string `json:"user_email"`
	UserId         string `json:"user_id"`
	UserProfileUrl string `json:"user_profile_url"`
}
*/

func main() {
	flag.Parse()

	go messageHub.run()

	router := gin.Default()

	if *serve != "" {
		router.LoadHTMLGlob(*serve + "/*")
		//homeTempl := template.Must(template.ParseGlob("/*"))
		servePath := "/resources/:page"
		log.Println("servePath:", servePath)
		router.GET(servePath, func(c *gin.Context) {
			page := c.Param("page")
			log.Println("Serve", page)
			c.HTML(http.StatusOK, page, gin.H{
				"Addr": c.Request.Host,
			})
			//log.Println(kami.Param(ctx, "page"))
			// homeTempl.Execute(w, r.Host)
			//homeTempl.ExecuteTemplate(w, kami.Param(ctx, "page"), r.Host)
		})
	}

	router.POST("/channel/:channel/event/chat_message/", ginChatHandler)

	router.GET("/socket/websocket", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})
	router.Run(*addr)
	//	kami.Serve()
}

func ginChatHandler(c *gin.Context) {
	body := c.Request.Body
	defer body.Close()
	requestBody, err := ioutil.ReadAll(body)
	if err != nil {
		//panic("ERROR: can't read http body")
		log.Println("ERROR: can't read http body")
		c.JSON(400, gin.H{"error": "can't read http body"})
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("requestBody:", string(requestBody))

	var message PostMessage
	c.Bind(&message)
	channel := c.Param("channel")
	channel = DefaultChat + "." + channel

	log.Printf("postHandler/channel => %s\n", channel)
	log.Printf("postHandler/message: %+v\n", message)

	nc := natsConnect()

	nc.Publish(channel, requestBody)
	c.JSON(200, gin.H{})
	//ncEnc, _ := nats.NewEncodedConn(nc, "json")
	//ncEnc.Publish(channel, &message)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// HandshakeTimeout specifies the duration for the handshake to complete.
	HandshakeTimeout: time.Second * 3,

	// CheckOrigin returns true if the request Origin header is acceptable. If
	// CheckOrigin is nil, the host in the Origin header must not be set or
	// must match the host of the request.
	//CheckOrigin func(r *http.Request) bool
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	if *Debug {
		log.Println("wsHandler fired")
	}
	hdr := make(http.Header)
	hdr["Access-Control-Allow-Origin"] = []string{"*"}
	ws, err := upgrader.Upgrade(w, r, hdr)
	// websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		return
	}

	// TODO: add here info about IP & UserAgent
	// log.Println("Create ws connection")
	// FIXME: move magic number to const
	c := &connection{
		send: make(chan []byte, 10), // limit queue to 10 messages
		ws:   ws,
		nc:   natsConnect(),
	}
	messageHub.register <- c
	defer func() { messageHub.unregister <- c }()
	go c.writer()
	c.reader()
}

func natsConnect() *nats.Conn {
	natsConn, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	return natsConn
}
