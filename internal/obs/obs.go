package obs

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "openmanus_http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"path", "method", "code"})
)

func Init() { prometheus.MustRegister(Requests) }

func AttachMetrics(r *gin.Engine) { r.GET("/metrics", gin.WrapH(promhttp.Handler())) }
