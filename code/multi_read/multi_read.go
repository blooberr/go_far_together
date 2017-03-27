package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type EsHealth struct {
	Cluster   string `json:"cluster"`
	NodeTotal string `json:"node.total"`
	Shards    string `json:"shards"`
}

func main() {
	// START OMIT
	var esResponse []EsHealth
	ticker := time.NewTicker(time.Millisecond * 500)
	go func() {
		for range ticker.C {
			resp, err := http.Get("http://localhost:8080/es/_cat/health?format=json&pretty")
			if err != nil {
				// handle error
				log.Printf("error fetching http %#v \n", err)
			}
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			json.Unmarshal(body, &esResponse)
			fmt.Printf("%#v \n", esResponse[0])
		}
	}()
	time.Sleep(time.Millisecond * 10000)
	ticker.Stop()
	// END OMIT
}
