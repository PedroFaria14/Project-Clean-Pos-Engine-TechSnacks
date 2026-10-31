package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterHealth exposes probes without business data or dependency diagnostics.
func RegisterHealth(router *gin.Engine, ping func(context.Context) error) {
	router.GET("/health/live", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "alive"}) })
	router.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not-ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
}
