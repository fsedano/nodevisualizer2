package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// It keeps a list of clients those are currently attached
// and broadcasting events to those clients.
type Event struct {
	// Events are pushed to this channel by the main events-gathering routine
	Message chan string

	// New client connections
	NewClients chan chan string

	// Closed client connections
	ClosedClients chan chan string

	// Total client connections
	TotalClients map[chan string]bool
}

// New event messages are broadcast to all registered client connection channels
type ClientChan chan string

func main() {
	router := gin.New()
	config := cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:    []string{"*"},
		ExposeHeaders:   []string{"*"},
		MaxAge:          12 * time.Hour,
	}

	router.Use(cors.New(config))
	router.Use(gin.Logger())

	// Initialize new streaming server
	stream := NewServer()

	// We are streaming current time to clients in the interval 10 seconds
	type dataNode struct {
		Id    string `json:"id"`
		Label string `json:"label"`
	}
	type dataEdge struct {
		Id     string `json:"id"`
		Source string `json:"source"`
		Target string `json:"target"`
	}
	type dataDAG struct {
		Nodes []dataNode `json:"nodes"`
		Edges []dataEdge `json:"edges"`
	}
	type dataRes struct {
		DAG dataDAG `json:"dag"`
	}
	type dataState struct {
		Resdata dataRes `json:"resdata"`
	}
	type dataFormat struct {
		Type      string    `json:"type"`
		StateData dataState `json:"state_data"`
	}
	go func() {
		for {
			time.Sleep(2 * time.Second)
			df := dataFormat{
				Type: "state",
				StateData: dataState{
					Resdata: dataRes{
						DAG: dataDAG{
							Edges: []dataEdge{
								{
									Source: "active",
									Target: "releasing",
								},
								{
									Source: "releasing",
									Target: "released",
								},

								{
									Source: "releasing",
									Target: "REL_NETWORK",
								},
								{
									Source: "releasing",
									Target: "POWER_OFF",
								},
								{
									Source: "releasing",
									Target: "POWER_OFF_VM",
								},

								{
									Source: "REL_NETWORK",
									Target: "POWER_OFF",
								},
								{
									Source: "POWER_OFF",
									Target: "POWER_OFF_VM",
								},
							},
							Nodes: []dataNode{
								{
									Label: "active",
									Id:    "active",
								},

								{
									Label: "releasing",
									Id:    "releasing",
								},
								{
									Label: "released",
									Id:    "released",
								},

								{
									Label: "REL_NETWORK",
									Id:    "REL_NETWORK",
								},
								{
									Label: "POWER_OFF",
									Id:    "POWER_OFF",
								},
								{
									Label: "POWER_OFF_VM",
									Id:    "POWER_OFF_VM",
								},
							},
						},
					},
				},
			}
			// pp, _ := json.MarshalIndent(df, "", "  ")
			// log.Printf("PP=%s", pp)
			b, _ := json.Marshal(df)
			dataToSend := fmt.Sprintf("%s", b)
			// Send current time to clients message channel
			stream.Message <- dataToSend
		}
	}()

	router.GET("/stream", HeadersMiddleware(), stream.serveHTTP(), func(c *gin.Context) {
		v, ok := c.Get("clientChan")
		if !ok {
			return
		}
		clientChan, ok := v.(ClientChan)
		if !ok {
			return
		}
		c.Stream(func(w io.Writer) bool {
			// Stream message to client from message channel
			if msg, ok := <-clientChan; ok {
				c.SSEvent("message", msg)
				return true
			}
			return false
		})
	})

	// Parse Static files
	router.StaticFile("/", "./public/index.html")

	router.Run(":8085")
}

// Initialize event and Start procnteessing requests
func NewServer() (event *Event) {
	event = &Event{
		Message:       make(chan string),
		NewClients:    make(chan chan string),
		ClosedClients: make(chan chan string),
		TotalClients:  make(map[chan string]bool),
	}

	go event.listen()

	return
}

// It Listens all incoming requests from clients.
// Handles addition and removal of clients and broadcast messages to clients.
func (stream *Event) listen() {
	for {
		select {
		// Add new available client
		case client := <-stream.NewClients:
			stream.TotalClients[client] = true
			log.Printf("Client added. %d registered clients", len(stream.TotalClients))

		// Remove closed client
		case client := <-stream.ClosedClients:
			delete(stream.TotalClients, client)
			close(client)
			log.Printf("Removed client. %d registered clients", len(stream.TotalClients))

		// Broadcast message to client
		case eventMsg := <-stream.Message:
			for clientMessageChan := range stream.TotalClients {
				clientMessageChan <- eventMsg
			}
		}
	}
}

func (stream *Event) serveHTTP() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Initialize client channel
		clientChan := make(ClientChan)

		// Send new connection to event server
		stream.NewClients <- clientChan

		defer func() {
			// Drain client channel so that it does not block. Server may keep sending messages to this channel
			go func() {
				for range clientChan {
				}
			}()
			// Send closed connection to event server
			stream.ClosedClients <- clientChan
		}()

		c.Set("clientChan", clientChan)

		c.Next()
	}
}

func HeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Next()
	}
}
