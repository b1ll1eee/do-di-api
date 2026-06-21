package router

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/b1ll1eee/flowdo-api/internal/adapter/inbound/http/handler"
	"github.com/b1ll1eee/flowdo-api/internal/adapter/inbound/http/middleware"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/inbound"
)

// New builds and returns the configured Gin engine.
// env should be "production" to enable release mode and suppress debug output.
// corsOrigins is the list of allowed origins, e.g. ["http://localhost:3000"].
// Pass ["*"] to allow all origins (development only).
func New(
	log zerolog.Logger,
	env string,
	corsOrigins []string,
	authSvc inbound.AuthService,
	flowdoHandler *handler.FlowdoHandler,
	authHandler *handler.AuthHandler,
) *gin.Engine {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()

	r.Use(corsMiddleware(corsOrigins))
	r.Use(requestLogger(log))
	r.Use(gin.Recovery())

	// Health-check — no auth required.
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Swagger UI.
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		// Public auth routes.
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Protected auth routes.
		authProtected := v1.Group("/auth", middleware.Auth(authSvc))
		{
			authProtected.GET("/me", authHandler.Me)
		}

		// Protected flowdo routes.
		flowdos := v1.Group("/flowdos", middleware.Auth(authSvc))
		{
			flowdos.POST("", flowdoHandler.Create)
			flowdos.GET("", flowdoHandler.List)
			flowdos.PATCH("/reorder", flowdoHandler.Reorder) // must be before /:id
			flowdos.GET("/:id", flowdoHandler.GetByID)
			flowdos.PUT("/:id", flowdoHandler.Update)
			flowdos.PATCH("/:id", flowdoHandler.Update)
			flowdos.DELETE("/:id", flowdoHandler.Delete)
		}
	}

	return r
}

// corsMiddleware returns a configured CORS middleware.
// When origins contains "*", all origins are permitted (suitable for development).
// In production, specify explicit origins only.
func corsMiddleware(origins []string) gin.HandlerFunc {
	cfg := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}

	// Wildcard shortcut — development only.
	for _, o := range origins {
		if o == "*" {
			cfg.AllowAllOrigins = true
			return cors.New(cfg)
		}
	}

	cfg.AllowOrigins = origins
	// Allow credentials only when specific origins are set (not wildcard).
	cfg.AllowCredentials = true
	return cors.New(cfg)
}

// requestLogger returns a Gin middleware that logs every request using zerolog.
func requestLogger(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		event := log.Info()
		if statusCode >= 500 {
			event = log.Error()
		} else if statusCode >= 400 {
			event = log.Warn()
		}

		if query != "" {
			path = path + "?" + query
		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Msg("request")
	}
}
