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

type TimeoutsConfig struct {
	CheckInterval time.Duration
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	DialTimeout   time.Duration
}

type Balancer interface {
	GetUpstreamProxy() *httputil.ReverseProxy
}
