package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync/atomic"
	"time"
)

func countRequests(c1, c2, c3 *int32) {
	go func() {
		t := time.NewTicker(time.Second * 5)
		for {
			select {
			case <-t.C:
				fmt.Printf("High: %v, Med: %v, Low:%v\n", atomic.LoadInt32(c1), atomic.LoadInt32(c2), atomic.LoadInt32(c3))
				atomic.StoreInt32(c1, 0)
				atomic.StoreInt32(c2, 0)
				atomic.StoreInt32(c3, 0)
			}
		}
	}()
}

func main() {
	portFlag := flag.String("port", "8081", "port")

	flag.Parse()

	s1 := http.NewServeMux()
	c1 := int32(0)
	c2 := int32(0)
	c3 := int32(0)

	s1.HandleFunc("/high", func(w http.ResponseWriter, r *http.Request) {
		for i := 0.0; i < 50000000; i++ {
			math.Sin(math.Cos(i))
		}
		atomic.AddInt32(&c1, 1)
	})

	s1.HandleFunc("/med", func(w http.ResponseWriter, r *http.Request) {
		for i := 0.0; i < 5000000; i++ {
			math.Cos(i)
		}
		atomic.AddInt32(&c2, 1)
	})

	s1.HandleFunc("/low", func(w http.ResponseWriter, r *http.Request) {
		for i := 0.0; i < 500; i++ {
			math.Cos(i)
		}
		atomic.AddInt32(&c3, 1)
	})

	countRequests(&c1, &c2, &c3)

	log.Printf("Run Test Server on :%v\n", *portFlag)
	log.Fatal(http.ListenAndServe(":"+*portFlag, s1))
}
