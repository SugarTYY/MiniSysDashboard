package api

import (
	"MiniSysDashboard/internal/collector"
	"MiniSysDashboard/internal/model"
	"MiniSysDashboard/internal/storage"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	collector *collector.Collector
	storage   *storage.Storage
	// SSE 客户端连接
	clients map[chan string]bool
	mu      sync.Mutex
}

func NewHandler(c *collector.Collector, s *storage.Storage) *Handler {
	return &Handler{
		collector: c,
		storage:   s,
		clients:   make(map[chan string]bool),
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// API 路由
	api := r.Group("/api")
	{
		api.GET("/metrics/current", h.GetCurrentMetrics)
		api.GET("/metrics/range", h.GetHistoryMetrics)
	}

	// SSE 路由
	r.GET("/sse/realtime", h.HandleSSE)
}

// StartCollection 启动后台采集循环
func (h *Handler) StartCollection(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			metrics, err := h.collector.Collect()
			if err != nil {
				log.Printf("Error collecting metrics: %v", err)
				continue
			}

			// 保存到存储（缓冲区）
			if err := h.storage.Save(metrics); err != nil {
				log.Printf("Error saving metrics: %v", err)
			}

			// 广播给 SSE 客户端
			h.broadcast(metrics)
		}
	}()
}

func (h *Handler) GetCurrentMetrics(c *gin.Context) {
	// 实时采集当前状态
	metrics, err := h.collector.Collect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (h *Handler) GetHistoryMetrics(c *gin.Context) {
	startStr := c.Query("start")
	endStr := c.Query("end")

	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start timestamp"})
		return
	}

	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end timestamp"})
		return
	}

	metrics, err := h.storage.QueryRange(start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

func (h *Handler) HandleSSE(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	clientChan := make(chan string)
	h.mu.Lock()
	h.clients[clientChan] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, clientChan)
		h.mu.Unlock()
		close(clientChan)
	}()

	c.Stream(func(w io.Writer) bool {
		if msg, ok := <-clientChan; ok {
			c.SSEvent("message", msg)
			return true
		}
		return false
	})
}

func (h *Handler) broadcast(metrics *model.Metrics) {
	data, err := json.Marshal(metrics)
	if err != nil {
		return
	}
	msg := string(data)

	h.mu.Lock()
	defer h.mu.Unlock()

	for clientChan := range h.clients {
		select {
		case clientChan <- msg:
		default:
			// 如果客户端太慢，丢弃消息
		}
	}
}
