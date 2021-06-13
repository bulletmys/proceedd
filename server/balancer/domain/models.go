package domain

import (
	"net/http/httputil"
	"time"
)

type HostsConfig struct {
	Host        string
	AppPort     int
	ServicePort int
	Weight      int
}

type UpstreamsConfig struct {
	CheckInterval time.Duration
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	DialTimeout   time.Duration
	WeightCoef    float64
	WeightType    int
	WeightMaxStep float64
}

type Balancer interface {
	GetUpstreamProxy() *httputil.ReverseProxy
}
