package api

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"PedroFaria14/clean-pos-engine/internal/domain/identity"

	"github.com/gin-gonic/gin"
)

const SessionCookieName = "techsnacks_session"
const AuthenticatedSessionKey = "authenticatedSession"

type AuthConfig struct {
	SecureCookies  bool
	AllowedOrigins []string
	TrustedProxies []string
}

func AuthConfigFromEnv() (AuthConfig, error) {
	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = "development"
	}
	if environment != "development" && environment != "production" {
		return AuthConfig{}, errors.New("APP_ENV must be development or production")
	}
	config := AuthConfig{SecureCookies: environment == "production"}
	origins := os.Getenv("APP_ALLOWED_ORIGINS")
	if origins == "" {
		if config.SecureCookies {
			return config, errors.New("APP_ALLOWED_ORIGINS is required in production")
		}
		origins = "http://localhost:5173,http://127.0.0.1:5173"
	}
	for _, origin := range strings.Split(origins, ",") {
		origin = strings.TrimSpace(origin)
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" || strings.Contains(u.Host, "*") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" ||
			(u.Scheme != "http" && u.Scheme != "https") || config.SecureCookies && u.Scheme != "https" {
			return config, errors.New("invalid allowed origin; production origins require HTTPS")
		}
		config.AllowedOrigins = append(config.AllowedOrigins, origin)
	}
	if value := os.Getenv("AUTH_TRUSTED_PROXIES"); value != "" {
		for _, proxy := range strings.Split(value, ",") {
			config.TrustedProxies = append(config.TrustedProxies, strings.TrimSpace(proxy))
		}
	}
	return config, nil
}

type AuthHandler struct {
	service *usecase.AuthService
	config  AuthConfig
}

func NewAuthHandler(service *usecase.AuthService, config AuthConfig) *AuthHandler {
	return &AuthHandler{service: service, config: config}
}

func Principal(c *gin.Context) (identity.Principal, bool) {
	value, exists := c.Get(AuthenticatedSessionKey)
	if !exists {
		return identity.Principal{}, false
	}
	session, ok := value.(usecase.AuthSession)
	return session.User, ok && session.User.ID != "" && (session.User.Role == identity.Operator || session.User.Role == identity.Admin)
}

func (h *AuthHandler) cookie(c *gin.Context, token string, expires time.Time, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{Name: SessionCookieName, Value: token, Path: "/", HttpOnly: true,
		Secure: h.config.SecureCookies, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: maxAge})
}

func authError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidCredentials), errors.Is(err, usecase.ErrUnauthenticated):
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials or session"})
	case errors.Is(err, usecase.ErrLoginLimited):
		c.Header("Retry-After", "900")
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many login attempts"})
	default:
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authentication temporarily unavailable"})
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	details := []gin.H{}
	if err := c.ShouldBindJSON(&input); err != nil {
		details = append(details, gin.H{"field": "body", "message": "invalid request body"})
	} else {
		if _, ok := identity.NormalizeUsername(input.Username); !ok {
			details = append(details, gin.H{"field": "username", "message": "use 3 to 64 ASCII letters, digits, dots, underscores or hyphens"})
		}
		if len(input.Password) < 1 || len(input.Password) > 128 {
			details = append(details, gin.H{"field": "password", "message": "password must contain 1 to 128 bytes"})
		}
	}
	if len(details) != 0 {
		c.JSON(400, gin.H{"error": "invalid login input", "details": details})
		return
	}
	session, token, err := h.service.Login(c.Request.Context(), input.Username, input.Password, c.ClientIP())
	if err != nil {
		authError(c, err)
		return
	}
	h.cookie(c, token, session.ExpiresAt, int(usecase.SessionLifetime.Seconds()))
	c.JSON(http.StatusCreated, session)
}

func (h *AuthHandler) RequireSession(c *gin.Context) {
	token, _ := c.Cookie(SessionCookieName)
	session, err := h.service.Authenticate(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthenticated) {
			h.cookie(c, "", time.Unix(1, 0), -1)
		}
		authError(c, err)
		return
	}
	c.Set(AuthenticatedSessionKey, session)
	c.Next()
}

func (h *AuthHandler) Current(c *gin.Context) {
	session, _ := c.Get(AuthenticatedSessionKey)
	c.JSON(http.StatusOK, session)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token, _ := c.Cookie(SessionCookieName)
	if err := h.service.Logout(c.Request.Context(), token); err != nil {
		authError(c, err)
		return
	}
	h.cookie(c, "", time.Unix(1, 0), -1)
	c.Status(http.StatusNoContent)
}

func RequireAdmin(c *gin.Context) {
	user, ok := Principal(c)
	if !ok || user.Role != identity.Admin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}
	c.Next()
}

func RequireCSRF(c *gin.Context) {
	value, ok := c.Get(AuthenticatedSessionKey)
	session, valid := value.(usecase.AuthSession)
	tokens := c.Request.Header.Values("X-CSRF-Token")
	if !ok || !valid || len(tokens) != 1 || session.CSRFToken == "" || subtle.ConstantTimeCompare([]byte(tokens[0]), []byte(session.CSRFToken)) != 1 {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid csrf token"})
		return
	}
	c.Next()
}

func (h *AuthHandler) SecurityPolicy(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Vary", "Origin")
	origin := c.GetHeader("Origin")
	allowed := false
	for _, candidate := range h.config.AllowedOrigins {
		if origin == candidate {
			allowed = true
			break
		}
	}
	unsafe := c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead
	if origin != "" && !allowed || unsafe && !allowed {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "origin not allowed"})
		return
	}
	if allowed {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, X-CSRF-Token")
	}
	if c.Request.Method == http.MethodOptions {
		c.AbortWithStatus(204)
		return
	}
	c.Next()
}

// NewRouter is the single production route registration point: deny anonymous business access.
func NewRouter(auth *AuthHandler, checkout *CheckoutHandler, products *ProductHandler, orders *OrderHandler) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(auth.config.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(func(c *gin.Context) {
		start := time.Now()
		id := rand.Text()
		c.Header("X-Request-Id", id)
		c.Next()
		slog.Info("HTTP request completed", "requestId", id, "method", c.Request.Method, "route", c.FullPath(), "status", c.Writer.Status(), "durationMs", time.Since(start).Milliseconds())
	})
	r.Use(gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, _ any) {
		slog.Error("HTTP request panicked", "requestId", c.Writer.Header().Get("X-Request-Id"))
		c.AbortWithStatusJSON(500, gin.H{"error": "internal server error"})
	}))
	r.Use(auth.SecurityPolicy)
	r.POST("/sessions", auth.Login)
	secured := r.Group("", auth.RequireSession)
	secured.GET("/sessions/current", auth.Current)
	secured.DELETE("/sessions/current", RequireCSRF, auth.Logout)
	secured.GET("/products", products.List)
	secured.POST("/products", RequireCSRF, RequireAdmin, products.Create)
	secured.PATCH("/products/:id", RequireCSRF, RequireAdmin, products.Update)
	secured.DELETE("/products/:id", RequireCSRF, RequireAdmin, products.Delete)
	secured.POST("/checkout", RequireCSRF, checkout.Handle)
	secured.GET("/orders/:id", orders.GetByID)
	return r, nil
}
