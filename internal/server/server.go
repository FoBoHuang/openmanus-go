package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"openmanus-go/internal/agent"
	"openmanus-go/internal/bus"
	"openmanus-go/internal/flow"
	"openmanus-go/internal/obs"
	"openmanus-go/internal/store"
)

type Server struct {
	Port  int
	Agent *agent.Agent
	Bus   *bus.Bus
	Store *store.Store
}

func New(port int, a *agent.Agent, b *bus.Bus, s *store.Store, enableMetrics bool, enablePProf bool) *Server {
	return &Server{Port: port, Agent: a, Bus: b, Store: s}
}

type runReq struct {
	Prompt         string      `json:"prompt"`
	Steps          []flow.Step `json:"steps"`
	TimeoutSeconds int         `json:"timeout_seconds"`
	Mode           string      `json:"mode"` // "steps" or "plan"
}

func (s *Server) router(enableMetrics bool) *gin.Engine {
	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	if enableMetrics {
		obs.Init()
		obs.AttachMetrics(r)
	}

	r.GET("/v1/tools", func(c *gin.Context) {
		ts := s.Agent.Tools.List()
		resp := make([]gin.H, 0, len(ts))
		for _, t := range ts {
			resp = append(resp, gin.H{"name": t.Name(), "desc": t.Desc(), "schema": t.Schema()})
		}
		c.JSON(200, resp)
	})

	r.POST("/v1/flow/run", func(c *gin.Context) {
		var req runReq
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		timeout := time.Duration(req.TimeoutSeconds) * time.Second
		if timeout == 0 {
			timeout = 60 * time.Second
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		runner := flow.NewRunner(s.Agent, s.Bus, s.Store)
		var out []flow.Result
		var err error
		if req.Mode == "plan" || (len(req.Steps) == 0 && req.Prompt != "") {
			out, err = runner.PlanAndRun(ctx, req.Prompt)
		} else {
			out, err = runner.Run(ctx, req.Steps)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"results": out})
	})

	return r
}

func (s *Server) Start(enableMetrics, enablePProf bool) error {
	addr := ":9000"
	if s.Port != 0 {
		addr = ":" + itoa(s.Port)
	}
	return s.router(enableMetrics).Run(addr)
}

func itoa(i int) string {
	x := i
	if x == 0 {
		return "0"
	}
	sign := ""
	if x < 0 {
		sign = "-"
		x = -x
	}
	buf := []byte{}
	for x > 0 {
		d := byte(x % 10)
		buf = append([]byte{d + '0'}, buf...)
		x /= 10
	}
	return sign + string(buf)
}
