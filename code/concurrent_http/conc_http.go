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
	ch <- fmt.Sprintf("%.2f elapsed with response length: %d %s", secs, len(body), url)
}

func main() {
	start := time.Now()
	ch := make(chan string)
	urls := []string{"http://www.google.com", "http://www.instacart.com", "http://apple.com", "http://www.yahoo.com"}
	for _, url := range urls {
		// concurrent HTTP request
		go MakeRequest(url, ch)
	}

	for range urls {
		fmt.Println(<-ch)
	}
	fmt.Printf("%.2fs elapsed\n", time.Since(start).Seconds())
}

// END OMIT
