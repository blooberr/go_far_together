package main

import (
	"bufio"
	"bytes"
	//"fmt"
	//"io/ioutil"
	"log"
	"net/http"
	"time"
)

func main() {
	var jsonStr = []byte(`
{ "index" : { "_index" : "test", "_type" : "type1", "_id" : "1" } }
{ "field1" : "value1" }
{ "delete" : { "_index" : "test", "_type" : "type1", "_id" : "2" } }
{ "create" : { "_index" : "test", "_type" : "type1", "_id" : "3" } }
{ "field1" : "value3" }
{ "update" : {"_id" : "1", "_type" : "type1", "_index" : "test"} }
{ "doc" : {"field2" : "value2"} }`)

	log.Printf("writing payload: %s\n", jsonStr)

	// START OMIT
	go func() {
		// listen on http stream
		resp, _ := http.Get("http://localhost:8001/stream")
		reader := bufio.NewReader(resp.Body)
		for {
			line, _ := reader.ReadBytes('\n')
			log.Println(string(line))
		}
	}()

	ticker := time.NewTicker(time.Millisecond * 500)
	go func() {
		for range ticker.C {
			// POST to this URL will write to multiple ES clusters
			url := "http://127.0.0.1:8080/es/_bulk?pretty"
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
			req.Header.Set("X-Custom-Header", "myvalue")
			req.Header.Set("Content-Type", "application/json")
			// END OMIT

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()
		}
	}()
	time.Sleep(time.Millisecond * 10000)
	ticker.Stop()
}
