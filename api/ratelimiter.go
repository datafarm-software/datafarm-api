package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/datafarm-software/datafarm-api/api/tokenprovider"
	"golang.org/x/time/rate"
)

type SessionLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var sessionLimiters = struct {
	sync.Mutex
	limitersMap map[string]SessionLimiter
}{limitersMap: make(map[string]SessionLimiter)}

func (a *Api) RateLimit(ctx huma.Context, next func(huma.Context)) {
	r, w := humamux.Unwrap(ctx)
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "No Authorization Header Provided.", http.StatusBadRequest)
		return
	}
	var lr tokenprovider.LoginResponse
	var key string
	parts := strings.Split(authHeader, "Bearer")
	if len(parts) == 2 {
		lr.Body = strings.TrimSpace(parts[1])
		lr.Body = strings.Trim(lr.Body, `"`)
		key = "token:" + lr.Body
	} else {
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
		key = "anon:" + ip
	}
	limiter := a.getLimiter(key)
	if !limiter.Allow() {
		http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
		return
	}
	next(ctx)
}

func (a *Api) getLimiter(key string) *rate.Limiter {
	sessionLimiters.Lock()
	defer sessionLimiters.Unlock()
	if v, exists := sessionLimiters.limitersMap[key]; exists {
		v.lastSeen = time.Now()
		sessionLimiters.limitersMap[key] = v
		return v.limiter
	}
	limiter := rate.NewLimiter(20, 50)
	sessionLimiters.limitersMap[key] = SessionLimiter{
		limiter:  limiter,
		lastSeen: time.Now(),
	}
	return limiter
}

func cleanupOldLimiters(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("limiter cleanup is stopping...")
			return
		case <-ticker.C:
			sessionLimiters.Lock()
			for id, v := range sessionLimiters.limitersMap {
				if time.Since(v.lastSeen) > time.Hour {
					delete(sessionLimiters.limitersMap, id)
				}
			}
			sessionLimiters.Unlock()
		}
	}
}
