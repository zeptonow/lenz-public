package middleware

import (
	"fmt"
	"net/http"
	"openreplay/backend/pkg/spot/keys"

	ctxStore "github.com/docker/distribution/context"

	"openreplay/backend/internal/config/common"
	"openreplay/backend/internal/http/util"
	"openreplay/backend/pkg/db/postgres/pool"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/metrics/database"
	"openreplay/backend/pkg/projects"
	"openreplay/backend/pkg/server/api"
	"openreplay/backend/pkg/server/auth"
	"openreplay/backend/pkg/server/limiter"
	"openreplay/backend/pkg/server/permissions"
	"openreplay/backend/pkg/server/tenant"
	"openreplay/backend/pkg/server/tracer"
	"openreplay/backend/pkg/server/user"
)

type baseMiddlewareBuilderImpl struct {
	middlewares []api.RouterMiddleware
}

func (b *baseMiddlewareBuilderImpl) Middlewares() []api.RouterMiddleware {
	return b.middlewares
}

func NewMiddlewareBuilder(
	log logger.Logger,
	jwtSecret string,
	interserviceKey string,
	http *common.HTTP,
	rtc *common.RateLimiter,
	prefix string,
	pgPool pool.Pool,
	dbMetric database.Database,
	handlers []api.Handlers,
	tenants tenant.Tenants,
	projects projects.Projects,
	extensionSecret *string,
	keys keys.Keys,
) (api.MiddlewareBuilder, error) {
	healthCheck := NewHealthCheck()
	corsCheck := NewCors(http.UseAccessControlHeaders)
	authenticator, err := auth.NewAuth(log, jwtSecret, interserviceKey, user.New(pgPool), tenants, projects, extensionSecret, keys, handlers)
	if err != nil {
		return nil, fmt.Errorf("error creating auth middleware: %s", err)
	}
	perms, err := permissions.NewPermissions(log, prefix, handlers)
	if err != nil {
		return nil, fmt.Errorf("error creating permissions middleware: %s", err)
	}
	rateLimiter, err := limiter.NewUserRateLimiter(rtc)
	if err != nil {
		return nil, fmt.Errorf("error creating rate limiter: %s", err)
	}
	audiTrail, err := tracer.NewTracer(log, pgPool, dbMetric, handlers)
	if err != nil {
		return nil, fmt.Errorf("error creating auditrail middleware: %s", err)
	}
	return &baseMiddlewareBuilderImpl{
		middlewares: []api.RouterMiddleware{healthCheck, corsCheck, NewSecurityHeaders(), authenticator, perms, rateLimiter, audiTrail},
	}, nil
}

func NewMinimalMiddlewareBuilder(http *common.HTTP) (api.MiddlewareBuilder, error) {
	return &baseMiddlewareBuilderImpl{
		middlewares: []api.RouterMiddleware{NewHealthCheck(), NewCors(http.UseAccessControlHeaders), NewSecurityHeaders(), NewIngestValidator()},
	}, nil
}

type healthCheckImpl struct{}

func NewHealthCheck() api.RouterMiddleware {
	return &healthCheckImpl{}
}

func (b *healthCheckImpl) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type corsImpl struct {
	useAccessControlHeaders bool
}

func NewCors(useAccessControlHeaders bool) api.RouterMiddleware {
	return &corsImpl{useAccessControlHeaders}
}

func (b *corsImpl) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if b.useAccessControlHeaders {
			// Prepare headers for preflight requests
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,Content-Encoding")
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Cache-Control", "max-age=86400")
			w.WriteHeader(http.StatusOK)
			return
		}

		r = r.WithContext(ctxStore.WithValues(r.Context(), map[string]interface{}{"httpMethod": r.Method, "url": util.SafeString(r.URL.Path)}))
		next.ServeHTTP(w, r)
	})
}

type securityHeadersImpl struct{}

func NewSecurityHeaders() api.RouterMiddleware {
	return &securityHeadersImpl{}
}

func (b *securityHeadersImpl) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers to prevent XSS, clickjacking, and enforce HTTPS
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'self';")
		next.ServeHTTP(w, r)
	})
}
