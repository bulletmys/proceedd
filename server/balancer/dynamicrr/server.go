package dynamicrr

import (
	"bufio"
	"fmt"
	"github.com/bulletmys/proceedd/server/balancer/domain"
	"log"
	"math"
	"net"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type ServerResources struct {
	cpuUtil int32
	memUsed int32
}

type Weight struct {
	weight1      int32
	weight2      int32
	startWeight  int32
	actualWeight int32
}

type Server struct {
	host        string
	servicePort int
	weight      Weight
	isAlive     atomic.Value
	lastCheckTS int64
	conn        net.Conn
	proxy       *httputil.ReverseProxy
	resources   ServerResources
}

func InitServers(hosts []domain.HostsConfig) ([]Server, error) {
	servers := make([]Server, 0, len(hosts))

	for _, val := range hosts {
		addr, err := url.Parse(fmt.Sprintf("http://%v:%v", val.Host, val.AppPort))
		if err != nil {
			return nil, fmt.Errorf("failed to parse server host: %w", err)
		}
		proxy := httputil.NewSingleHostReverseProxy(addr)

		w := Weight{}
		w.setWeight(int32(val.Weight))
		server := Server{
			host:        val.Host,
			servicePort: val.ServicePort,
			weight:      w,
			proxy:       proxy,
		}
		server.isAlive.Store(0)
		servers = append(servers, server)
	}

	return servers, nil
}

// setWeight set new weight for server
func (s *Weight) setWeight(newWeight int32) {
	if atomic.LoadInt32(&s.actualWeight) == 0 {
		atomic.StoreInt32(&s.weight2, newWeight)
		atomic.StoreInt32(&s.actualWeight, 1)
		return
	}
	atomic.StoreInt32(&(s.weight1), newWeight)
	atomic.StoreInt32(&(s.actualWeight), 0)
}

// getWeight return actual weight of server
func (s *Weight) getWeight() int32 {
	if atomic.LoadInt32(&s.actualWeight) == 0 {
		return atomic.LoadInt32(&s.weight1)
	}
	return atomic.LoadInt32(&s.weight2)
}

// getStartWeight returns start weight of server
func (s *Weight) getStartWeight() int32 {
	return s.startWeight
}

// getBothWeights return actual and previous weight of server
func (s *Weight) getBothWeights() (actual, previous int32) {
	if atomic.LoadInt32(&s.actualWeight) == 0 {
		return atomic.LoadInt32(&s.weight1), atomic.LoadInt32(&s.weight2)
	}
	return atomic.LoadInt32(&s.weight2), atomic.LoadInt32(&s.weight1)
}

func (s *Server) updateWeight(averageCpu, averageMem int32, weightCoef float64, wType int, wStep float64) {
	weight := float64(s.weight.getWeight())

	cpuUtil := atomic.LoadInt32(&s.resources.cpuUtil)
	memUsed := atomic.LoadInt32(&s.resources.memUsed)

	cpuCoef := float64(averageCpu) / (float64(cpuUtil) + 0.001)
	memCoef := float64(averageMem) / (float64(memUsed) + 0.001)

	var newWeight float64

	switch wType {
	case 1:
		newWeight = math.Round(weight * cpuCoef * memCoef * weightCoef)
	case 2:
		newWeight = math.Round(float64(s.weight.getStartWeight()) * math.Pow((cpuCoef+memCoef)/2, weightCoef))
	case 3:
		wCoef := math.Abs(float64(s.weight.getStartWeight())/weight - 1)
		newWeight = math.Round(weight * math.Pow((cpuCoef+memCoef+wCoef)/3, weightCoef))
	}

	lowBound := math.Round(weight * (1.0 - wStep))
	highBound := math.Round(weight * (1.0 + wStep))

	if newWeight > highBound {
		newWeight = highBound
	}

	if newWeight < lowBound {
		newWeight = lowBound
	}

	if newWeight > 1 {
		s.weight.setWeight(int32(newWeight))
	}

	fmt.Printf("%v:%v prevW: %v. actualW: %v\n"+
		"avgcpu: %v, avgmem: %v, cpu: %v mem: %v, cpuCoef: %v, memCoef: %v\n\n",
		s.host, s.servicePort, weight, newWeight, averageCpu, averageMem, cpuUtil, memUsed, cpuCoef, memCoef,
	)
}

func (s *ServerResources) parseResources(data string) error {
	indicators := strings.Split(strings.TrimRight(data, "\n"), ":")

	cpu, err1 := strconv.Atoi(indicators[0])
	mem, err2 := strconv.Atoi(indicators[1])
	if err1 != nil || err2 != nil {
		return fmt.Errorf("failed to parse resources. cpu: %v, mem: %v", err1, err2)
	}

	atomic.StoreInt32(&s.cpuUtil, int32(cpu))
	atomic.StoreInt32(&s.memUsed, int32(mem))
	return nil
}

func (s *Server) getServerResources(writeTimeout, readTimeout time.Duration) (string, error) {
	if s.conn == nil {
		return "", fmt.Errorf("nil connection to server: %v:%v", s.host, s.servicePort)
	}
	if err := s.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return "", fmt.Errorf("failed to set write deadline timeout: %w\n", err)
	}
	_, err := fmt.Fprintf(s.conn, "stat\n")
	if err != nil { //todo retries
		return "", err
	}

	if err := s.conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return "", fmt.Errorf("failed to set read deadline timeout: %w\n", err)
	}
	data, err := bufio.NewReader(s.conn).ReadString('\n')
	if err != nil { //todo retries
		return "", err
	}

	s.lastCheckTS = time.Now().Unix()

	return data, nil
}

func (s *Server) checkServer(cpuUtil, memUsed int32, cfg domain.UpstreamsConfig) {
	if s.conn == nil || (s.conn != nil && s.isAlive.Load() == 0) {
		if s.conn != nil {
			s.conn.Close()
		}
		conn, err := net.DialTimeout(
			"tcp", fmt.Sprintf("%v:%v", s.host, s.servicePort), cfg.DialTimeout,
		)
		if err != nil {
			s.isAlive.Store(0)
			log.Printf("failed to connect to upstream server: %v\n", err)
			return
		}
		log.Printf("connection established to upstream server: %v\n", conn.RemoteAddr())
		s.conn = conn
	}
	data, err := s.getServerResources(cfg.WriteTimeout, cfg.ReadTimeout)
	if err != nil {
		s.isAlive.Store(0)
		log.Printf("failed to check server %v:%v, err: %v", s.host, s.servicePort, err)
		return
	}
	if data != "" {
		log.Printf("Host: %v, Data: %v", s.conn.RemoteAddr(), data)
		if err := s.resources.parseResources(data); err != nil {
			s.isAlive.Store(0)
			log.Printf("%v", err)
			return
		}
		s.updateWeight(cpuUtil, memUsed, cfg.WeightCoef, cfg.WeightType, cfg.WeightMaxStep)
		s.isAlive.Store(1)
	}
}
