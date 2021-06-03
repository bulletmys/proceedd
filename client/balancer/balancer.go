package balancer

import (
	"bufio"
	"fmt"
	"github.com/golobby/config/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

type stat struct {
	cpuUsage   int32
	memUsage   int32
}

func (s *stat) updateStats() {
	for {
		memStats, errMem := mem.VirtualMemory()
		cpuStat, errCpu := cpu.Percent(0, false)
		if errCpu != nil || errMem != nil {
			log.Printf("failed to get stats – cpu: %v, mem: %v", errCpu, errMem)
		}
		atomic.StoreInt32(&s.cpuUsage, int32(cpuStat[0]))
		atomic.StoreInt32(&s.memUsage, int32(memStats.UsedPercent))
		time.Sleep(5*time.Second)
	}
}

func (s *stat) handleConnection(c net.Conn) {
	log.Printf("Serving %s\n", c.RemoteAddr().String())
	for {
		netData, err := bufio.NewReader(c).ReadString('\n')
		log.Printf("Readed")
		if err != nil {
			log.Println(err)
			break
		}
		temp := strings.TrimSpace(netData)
		if temp == "STOP" {
			break
		}
		log.Printf("received: %v", temp)

		statData := []byte(fmt.Sprintf("%v:%v\n",
			atomic.LoadInt32(&s.cpuUsage),
			atomic.LoadInt32(&s.memUsage),
		))
		if _, err = c.Write(statData); err != nil {
			log.Printf("failed to write to tcp connection: %v", err)
		}
	}
	log.Printf("Closing %s\n", c.RemoteAddr().String())
	c.Close()
}

func Start(c *config.Config) {
	port, err := c.GetString("balancer.server_port")
	if err != nil {
		log.Fatalf("failed to get port from config: %v", err)
	}

	var s stat

	go s.updateStats()

	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to start listening tcp: %v", err)
	}
	defer l.Close()

	log.Printf("balancer client start listening on port %v", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("failed to accept conn: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}
