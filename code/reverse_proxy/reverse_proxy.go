package main

/*
  Do NOT use for production.
  Demo code for presentation. Use at own risk.
  - Jon
*/
import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	//"time"

	"gopkg.in/gin-gonic/gin.v1"
)

type Broker struct {

	// Events are pushed to this channel by the main events-gathering routine
	Notifier chan []byte

	// New client connections
	newClients chan chan []byte

	// Closed client connections
	closingClients chan chan []byte

	// Client connections registry
	clients map[chan []byte]bool
}

func NewServer() (broker *Broker) {
	// Instantiate a broker
	broker = &Broker{
		Notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
	}

	// Set it running - listening and broadcasting events
	go broker.listen()

	return
}

func (broker *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	// Make sure that the writer supports flushing.
	//
	flusher, ok := rw.(http.Flusher)

	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	// Each connection registers its own message channel with the Broker's connections registry
	messageChan := make(chan []byte)

	// Signal the broker that we have a new connection
	broker.newClients <- messageChan

	// Remove this client from the map of connected clients
	// when this handler exits.
	defer func() {
		broker.closingClients <- messageChan
	}()

	// Listen to connection close and un-register messageChan
	notify := rw.(http.CloseNotifier).CloseNotify()

	go func() {
		<-notify
		broker.closingClients <- messageChan
	}()

	for {

		// Write to the ResponseWriter
		// Server Sent Events compatible
		fmt.Fprintf(rw, "data: %s\n\n", <-messageChan)

		// Flush the data immediatly instead of buffering it for later.
		flusher.Flush()
	}

}

func (broker *Broker) listen() {
	for {
		select {
		case s := <-broker.newClients:

			// A new client has connected.
			// Register their message channel
			broker.clients[s] = true
			log.Printf("Client added. %d registered clients", len(broker.clients))
		case s := <-broker.closingClients:

			// A client has dettached and we want to
			// stop sending them messages.
			delete(broker.clients, s)
			log.Printf("Removed client. %d registered clients", len(broker.clients))
		case event := <-broker.Notifier:

			// We got a new event from the outside!
			// Send event to all connected clients
			for clientMessageChan, _ := range broker.clients {
				clientMessageChan <- event
			}
		}
	}

}

type myTransport struct {
	// Uncomment this if you want to capture the transport
	//CapturedTransport http.RoundTripper
}

type EsNode struct {
	Url   string `json:"url" binding:"required"`
	Read  bool   `json:"read"`
	Write bool   `json:"write"`
}

func CreateEsNode(url string) *EsNode {
	return &EsNode{
		Url:   url,
		Read:  false,
		Write: false,
	}
}

type CommandPacket struct {
	Command string
	Url     string
}

type CommandResponse struct {
	Command string   `json:"command"`
	EsNode  *EsNode  `json:"es_node"`
	URLs    []string `json:"urls"`
}

func (t *myTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	log.Printf("request: %#v \n", request)
	response, err := http.DefaultTransport.RoundTrip(request)
	log.Printf("RoundTrip: response: %#v \n", response)
	body, err := httputil.DumpResponse(response, true)
	if err != nil {
		return nil, err
	}
	log.Printf("body: %#v", string(body))

	return response, err
}

func ReverseProxy(cpChan chan<- *CommandPacket, responseChan <-chan *CommandResponse) gin.HandlerFunc {
	return func(c *gin.Context) {
		cpChan <- &CommandPacket{Command: "readable_nodes"}
		response := <-responseChan
		urls := response.URLs
		log.Printf("readable URLs: %#v \n", urls)
		log.Printf("index: %d \n", rand.Intn(len(urls)))
		urlTarget := urls[rand.Intn(len(urls))]

		log.Printf("choosing %s \n", urlTarget)

		director := func(req *http.Request) {
			s, _ := url.Parse(urlTarget)
			r := c.Request
			req = r
			req.URL.Scheme = "http"
			log.Printf("host: %s \n", s.Host)
			log.Printf("s: %#v \n", s)
			log.Printf("r: %#v \n", r.URL.Path)
			paths := strings.Split(r.URL.Path, "/es")
			req.URL.Path = paths[len(paths)-1]
			req.URL.Host = s.Host
			req.Header["my-header"] = []string{r.Header.Get("my-header")}
			delete(req.Header, "My-Header")
		}
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(c.Writer, c.Request)
		return
	}
}

func Coordinator(cp <-chan *CommandPacket, responseChan chan<- *CommandResponse) {

	esNodes := make(map[string]*EsNode)

	for req := range cp {
		switch req.Command {
		case "writeable_nodes":
			var writeables []string
			for url, node := range esNodes {
				if node.Write == true {
					writeables = append(writeables, url)
				}
			}
			cr := &CommandResponse{Command: "writeable_nodes", URLs: writeables}
			responseChan <- cr
		case "readable_nodes":
			var writeables []string
			for url, node := range esNodes {
				if node.Read == true {
					writeables = append(writeables, url)
				}
			}
			cr := &CommandResponse{Command: "readablee_nodes", URLs: writeables}
			responseChan <- cr
			// START CODEMAPOMIT
		case "add_node":
			esNodes[req.Url] = CreateEsNode(req.Url)
			responseChan <- &CommandResponse{EsNode: esNodes[req.Url]}
		case "allow_read":
			if node := esNodes[req.Url]; node != nil {
				node.Read = true
				responseChan <- &CommandResponse{EsNode: esNodes[req.Url]}
				break
			}
			responseChan <- &CommandResponse{}
		case "allow_write":
			if node := esNodes[req.Url]; node != nil {
				node.Write = true
				responseChan <- &CommandResponse{EsNode: esNodes[req.Url]}
				break
			}
			responseChan <- &CommandResponse{}
			// END CODEMAPOMIT
		case "remove_node":
			fmt.Printf("removing node...")
		}
	}
}

func MassWrite(cpChan chan<- *CommandPacket, responseChan <-chan *CommandResponse, eventChan chan<- string) gin.HandlerFunc {
	cpChan <- &CommandPacket{Command: "writeable_nodes"}
	response := <-responseChan
	urls := response.URLs

	return func(c *gin.Context) {
		reuse := c.Request
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
		}

		// Restore the io.ReadCloser to its original state
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		for _, target := range urls {
			director := func(req *http.Request) {
				s, _ := url.Parse(target)
				r := c.Request
				req = r
				req.URL.Scheme = "http"
				req.URL.Host = s.Host
				req.Header["my-header"] = []string{r.Header.Get("my-header")}

				c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
				delete(req.Header, "My-Header")
			}
			proxy := &httputil.ReverseProxy{Director: director}
			proxy.Transport = &myTransport{}
			proxy.ServeHTTP(c.Writer, reuse)
		}

		msg := fmt.Sprintf("Wrote to %d nodes!\r\n", len(urls))
		eventChan <- msg
	}
}

func WebServer(cpChan chan<- *CommandPacket, responseChan <-chan *CommandResponse, eventChan chan<- string) {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// START COFFEEOMIT
	r.POST("/node", func(c *gin.Context) {
		var esNode EsNode
		if c.BindJSON(&esNode) == nil {
			cpChan <- &CommandPacket{Command: "add_node", Url: esNode.Url}
			response := <-responseChan
			c.JSON(200, gin.H{"status": response})
		} else {
			c.JSON(500, gin.H{"status": "invalid request"})
		}
	})

	r.GET("/nodes/writeable", func(c *gin.Context) {
		cpChan <- &CommandPacket{Command: "writeable_nodes"}
		response := <-responseChan
		c.JSON(200, response)
	})

	r.POST("/es/_bulk", MassWrite(cpChan, responseChan, eventChan))
	r.GET("/es/_cat/health", ReverseProxy(cpChan, responseChan))

	r.Run() // listen and serve on 0.0.0.0:8080
	// END COFFEEOMIT
}

func main() {
	cpChan := make(chan *CommandPacket)
	responseChan := make(chan *CommandResponse)
	eventChan := make(chan string)

	go func() {
		Coordinator(cpChan, responseChan)
	}()

	go func() {
		broker := NewServer()

		go func() {
			for message := range eventChan {
				broker.Notifier <- []byte(message)
			}
		}()

		log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:8001", broker))

	}()

  // adding nodes, enabling read/writes
	es := CreateEsNode("http://localhost:9200")
	log.Printf("es: %#v \n", es)

	cpChan <- &CommandPacket{Command: "add_node", Url: "http://localhost:9200"}
	response := <-responseChan
	log.Printf("add response: %#v \n", response)

	cpChan <- &CommandPacket{Command: "allow_read", Url: "http://localhost:9200"}
	response = <-responseChan
	log.Printf("response: %#v \n", response)

	cpChan <- &CommandPacket{Command: "allow_write", Url: "http://localhost:9200"}
	response = <-responseChan
	log.Printf("response: %#v \n", response)

	cpChan <- &CommandPacket{Command: "add_node", Url: "http://localhost:9801"}
	response = <-responseChan
	log.Printf("add response: %#v \n", response)

	cpChan <- &CommandPacket{Command: "allow_read", Url: "http://localhost:9801"}
	response = <-responseChan
	log.Printf("response: %#v \n", response)

	cpChan <- &CommandPacket{Command: "allow_write", Url: "http://localhost:9801"}
	response = <-responseChan
	log.Printf("response: %#v \n", response)

	WebServer(cpChan, responseChan, eventChan)

}
