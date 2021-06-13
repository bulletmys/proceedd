package dynamicrr

import (
	"github.com/bulletmys/proceedd/server/balancer/domain"
	"log"
	"net/http/httputil"
	"sync/atomic"
	"time"
)

type Upstreams struct {
	servers          []Server
	averageResources ServerResources
	cfg              domain.UpstreamsConfig
	done             <-chan struct{}
	proxyChan        chan *Server
}

func NewUpstreams(timeouts domain.UpstreamsConfig, hosts []domain.HostsConfig, done <-chan struct{}) (*Upstreams, error) {
	servers, err := InitServers(hosts)
	if err != nil {
		return nil, err
	}

	ups := &Upstreams{
		servers:   servers,
		cfg:       timeouts,
		done:      done,
		proxyChan: make(chan *Server, 5),
	}

	log.Println("run server updater")
	go ups.update()

	// wait for init
	time.Sleep(timeouts.CheckInterval)

	log.Println("run avg resource updater")
	go ups.updateAverageResources()

	time.Sleep(2 * time.Second)

	log.Println("run round robin")
	go ups.roundRobin()

	return ups, nil
}

func (u *Upstreams) wrapServerChecker(s *Server) {
	go func() {
		ticker := time.NewTicker(u.cfg.CheckInterval)

		for {
			select {
			case <-u.done:
				return
			case <-ticker.C:
				s.checkServer(
					atomic.LoadInt32(&u.averageResources.cpuUtil),
					atomic.LoadInt32(&u.averageResources.memUsed),
					u.cfg,
				)
			}
		}
	}()
}

func (u *Upstreams) update() {
	for i := range u.servers {
		u.wrapServerChecker(&u.servers[i])
	}
}

func (u *Upstreams) updateAverageResources() {
	for {
		aliveCount := int32(0)
		cpuSum := int32(0)
		memSum := int32(0)

		for i := range u.servers {
			if u.servers[i].isAlive.Load() == 0 {
				continue
			}
			aliveCount++
			cpuSum += atomic.LoadInt32(&u.servers[i].resources.cpuUtil)
			memSum += atomic.LoadInt32(&u.servers[i].resources.memUsed)
		}
		if aliveCount == 0 {
			log.Println("CRITICAL: no alive servers")
		} else {
			atomic.StoreInt32(&u.averageResources.cpuUtil, cpuSum/aliveCount)
			atomic.StoreInt32(&u.averageResources.memUsed, memSum/aliveCount)
		}
		time.Sleep(u.cfg.CheckInterval/2)
	}
}

func (u *Upstreams) roundRobin() {
	isEmptyRound := false
	round := int32(0)

	for {
		for !isEmptyRound {
			isEmptyRound = true

			for i := range u.servers {
				if u.servers[i].isAlive.Load() == 0 {
					continue
				}
				if u.servers[i].weight.getWeight()-round < 1 {
					continue
				}
				isEmptyRound = false

				u.proxyChan <- &u.servers[i]
			}
			round++
		}
		round = 0
		isEmptyRound = false
	}
}

func (u *Upstreams) GetUpstreamProxy() *httputil.ReverseProxy {
	s := <-u.proxyChan
	if s.isAlive.Load() == 1 {
		return s.proxy
	}
	return u.GetUpstreamProxy()
}
