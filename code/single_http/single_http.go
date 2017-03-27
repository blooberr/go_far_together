// concurrent.go
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// START OMIT
func MakeRequest(url string, ch chan<- string) {
	start := time.Now()
	resp, _ := http.Get(url)

	secs := time.Since(start).Seconds()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("%.2f elapsed with response length: %d %s\n", secs, len(body), url)
}

func main() {
	start := time.Now()
	ch := make(chan string)
	urls := []string{"http://www.google.com", "http://www.instacart.com", "http://apple.com", "http://www.yahoo.com"}
	for _, url := range urls {
		MakeRequest(url, ch)
	}

	fmt.Printf("%.2fs elapsed\n", time.Since(start).Seconds())
}
// END OMIT
