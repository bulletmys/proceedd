package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

func countRequests(c1, c2 *int32) {
	go func() {
		t := time.NewTicker(time.Second * 5)
		for {
			select {
			case <-t.C:
				fmt.Printf("Server1: %v, Server2: %v\n", atomic.LoadInt32(c1), atomic.LoadInt32(c2))
				atomic.StoreInt32(c1, 0)
				atomic.StoreInt32(c2, 0)
			}
		}
	}()
}

func main() {
	s1 := http.NewServeMux()
	s2 := http.NewServeMux()
	counter1 := int32(0)
	counter2 := int32(0)

	s1.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "1 -- Server -- 1")
		atomic.AddInt32(&counter1, 1)
	})
	s2.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "2 -- Server -- 2")
		atomic.AddInt32(&counter2, 1)
	})

	countRequests(&counter1, &counter2)

	go func() {
		log.Fatal(http.ListenAndServe(":8081", s1))
	}()
	log.Fatal(http.ListenAndServe(":8082", s2))
}
