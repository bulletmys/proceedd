package balancer

import (
	"fmt"
	"github.com/bulletmys/proceedd/server/balancer/domain"
	"github.com/bulletmys/proceedd/server/balancer/dynamicrr"
	"github.com/golobby/config/v2"
	"log"
	"net/http"
	"net/http/pprof"
	_ "net/http/pprof"
	"time"
)

type Proxy struct {
	balancer domain.Balancer
}

func initConfigForDynamicRR(c *config.Config) (domain.UpstreamsConfig, []domain.HostsConfig, int) {
	upsAddr, err := c.Get("balancer.upstreams")
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}
	upstreamsAddrInterface, ok := upsAddr.([]interface{})
	if !ok {
		log.Fatalf("failed to cast upstreams to array")
	}

	upstreamsAddr := make([]domain.HostsConfig, len(upstreamsAddrInterface))

	for i, val := range upstreamsAddrInterface {
		v, ok := val.(map[interface{}]interface{})
		if !ok {
			log.Fatalln("failed to convert server config")
		}

		host, ok1 := v["host"].(string)
		appPort, ok2 := v["app_port"].(int)
		servicePort, ok3 := v["service_port"].(int)
		weight, ok4 := v["weight"].(int) //todo check if greater than 0
		if !ok1 || !ok2 || !ok3 || !ok4 {
			log.Fatalln("failed to convert server config")
		}
		upstreamsAddr[i] = domain.HostsConfig{Host: host, AppPort: appPort, ServicePort: servicePort, Weight: weight}
	}

	checkInterval, err := c.GetString("balancer.check_interval")
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}
	checkIntervalDuration, err := time.ParseDuration(checkInterval)
	if err != nil {
		log.Fatalf("failed to parse check_interval: %v", err)
	}

	readTimeout, err := c.GetString("balancer.read_timeout")
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}
	readTimeoutDuration, err := time.ParseDuration(readTimeout)
	if err != nil {
		log.Fatalf("failed to parse read_timeout: %v", err)
	}

	writeTimeout, err := c.GetString("balancer.write_timeout")
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}
	writeTimeoutDuration, err := time.ParseDuration(writeTimeout)
	if err != nil {
		log.Fatalf("failed to parse write_timeout: %v", err)
	}

	dialTimeout, err := c.GetString("balancer.dial_timeout")
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}
	dialTimeoutDuration, err := time.ParseDuration(dialTimeout)
	if err != nil {
		log.Fatalf("failed to parse dial_timeout: %v", err)
	}

	port, err := c.GetInt("balancer.port")
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	weightCoef, err := c.GetFloat("balancer.weight_coef")
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	wType, err := c.GetInt("balancer.weight_type")
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	wMaxStep, err := c.GetFloat("balancer.weight_max_step")
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	return domain.UpstreamsConfig{
		CheckInterval: checkIntervalDuration,
		ReadTimeout:   readTimeoutDuration,
		WriteTimeout:  writeTimeoutDuration,
		DialTimeout:   dialTimeoutDuration,
		WeightCoef:    weightCoef,
		WeightType:    wType,
		WeightMaxStep: wMaxStep,
	}, upstreamsAddr, port

}

func Start(c *config.Config) {
	cfg, hosts, port := initConfigForDynamicRR(c)

	done := make(chan struct{})
	defer close(done)

	ups, err := dynamicrr.NewUpstreams(cfg, hosts, done)
	if err != nil {
		log.Fatalf("failed to init upstreams: %v", err)
	}
	proxy := Proxy{ups}

	s := http.NewServeMux()
	s.HandleFunc("/", proxy.handler())

	// Register pprof handlers
	s.HandleFunc("/debug/pprof/", pprof.Index)
	s.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	s.HandleFunc("/debug/pprof/profile", pprof.Profile)
	s.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	s.HandleFunc("/debug/pprof/trace", pprof.Trace)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), s))
}

func (u *Proxy) handler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL)
		p := u.balancer.GetUpstreamProxy()
		p.ServeHTTP(w, r)
	}
}
