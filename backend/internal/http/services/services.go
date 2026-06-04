package services

import (
	"errors"
	"openreplay/backend/internal/http/uaparser"

	apiConfig "openreplay/backend/internal/config/api"
	"openreplay/backend/internal/config/http"
	"openreplay/backend/internal/http/geoip"
	"openreplay/backend/pkg/assist/proxy"
	"openreplay/backend/pkg/canvas"
	"openreplay/backend/pkg/conditions"
	conditionsAPI "openreplay/backend/pkg/conditions/api"
	"openreplay/backend/pkg/db/postgres/pool"
	"openreplay/backend/pkg/db/redis"
	"openreplay/backend/pkg/events"
	eventAPI "openreplay/backend/pkg/events/api"
	"openreplay/backend/pkg/flakeid"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/metrics/database"
	"openreplay/backend/pkg/metrics/web"
	"openreplay/backend/pkg/objectstorage/store"
	"openreplay/backend/pkg/projects"
	"openreplay/backend/pkg/queue/types"
	replayAPI "openreplay/backend/pkg/replays/api"
	"openreplay/backend/pkg/replays/service"
	sdkAPI "openreplay/backend/pkg/sdk/api"
	"openreplay/backend/pkg/server/api"
	"openreplay/backend/pkg/session"
	sessionAPI "openreplay/backend/pkg/session/api"
	"openreplay/backend/pkg/sessions"
	mobilesessions "openreplay/backend/pkg/sessions/api/mobile"
	websessions "openreplay/backend/pkg/sessions/api/web"
	"openreplay/backend/pkg/tags"
	tagsAPI "openreplay/backend/pkg/tags/api"
	"openreplay/backend/pkg/token"
)

// noOpAssist is a no-op implementation of the Assist interface
// Used when live session (assist) feature is not enabled
type noOpAssist struct{}

func NewNoOpAssist() proxy.Assist {
	return &noOpAssist{}
}

func (n *noOpAssist) GetLiveSessionByID(projID uint32, sessID uint64) (interface{}, error) {
	return nil, errors.New("live sessions not available")
}

func (n *noOpAssist) GetLiveSessionsWS(projID uint32, req *proxy.GetLiveSessionsRequest) (interface{}, error) {
	return nil, errors.New("live sessions not available")
}

func (n *noOpAssist) IsLive(projID uint32, sessID uint64) (bool, error) {
	return false, nil
}

// newSessionConfigAdapter creates a minimal api.Config from http.Config
// Only populates fields needed by session API handlers
func newSessionConfigAdapter(cfg *http.Config) *apiConfig.Config {
	return &apiConfig.Config{
		HTTP: cfg.HTTP,
	}
}

type serviceBuilder struct {
	webAPI        api.Handlers
	mobileAPI     api.Handlers
	conditionsAPI api.Handlers
	tagsAPI       api.Handlers
	sdkAPI        api.Handlers
	replayAPI     api.Handlers
	sessionAPI    api.Handlers
	eventAPI      api.Handlers
}

func (b *serviceBuilder) Handlers() []api.Handlers {
	return []api.Handlers{b.webAPI, b.mobileAPI, b.conditionsAPI, b.tagsAPI, b.sdkAPI, b.replayAPI, b.sessionAPI, b.eventAPI}
}

func New(log logger.Logger, cfg *http.Config, webMetrics web.Web, dbMetrics database.Database, producer types.Producer, pgconn pool.Pool, redis *redis.Client) (api.ServiceBuilder, error) {
	projs := projects.New(log, pgconn, redis, dbMetrics)
	geoModule, err := geoip.New(log, cfg.MaxMinDBFile)
	if err != nil {
		return nil, err
	}
	uaModule, err := uaparser.NewUAParser(cfg.UAParserFile)
	if err != nil {
		return nil, err
	}
	tokenizer := token.NewTokenizer(cfg.TokenSecret)
	conditions := conditions.New(pgconn)
	flaker := flakeid.NewFlaker(cfg.WorkerID)
	sessions := sessions.New(log, pgconn, projs, redis, dbMetrics)
	tags := tags.New(log, pgconn)
	responser := api.NewResponser(webMetrics)

	// Initialize dependencies for replay handlers
	objStore, err := store.NewStore(&cfg.ObjectsConfig)
	if err != nil {
		return nil, err
	}

	canvases, err := canvas.New(log, pgconn, dbMetrics)
	if err != nil {
		return nil, err
	}

	files, err := service.New(log, cfg, objStore, canvases)
	if err != nil {
		return nil, err
	}

	sessionService, err := session.NewService(log, pgconn, nil, files)
	if err != nil {
		return nil, err
	}

	builder := &serviceBuilder{}
	if builder.webAPI, err = websessions.NewHandlers(cfg, log, responser, producer, projs, sessions, uaModule, geoModule, tokenizer, conditions, flaker); err != nil {
		return nil, err
	}
	if builder.mobileAPI, err = mobilesessions.NewHandlers(cfg, log, responser, producer, projs, sessions, geoModule, tokenizer, conditions, flaker); err != nil {
		return nil, err
	}
	if builder.conditionsAPI, err = conditionsAPI.NewHandlers(log, responser, tokenizer, conditions); err != nil {
		return nil, err
	}
	if builder.tagsAPI, err = tagsAPI.NewHandlers(log, responser, tokenizer, sessions, tags); err != nil {
		return nil, err
	}
	if builder.sdkAPI, err = sdkAPI.NewHandlers(cfg, log, responser, producer, projs, sessions, geoModule, tokenizer, flaker); err != nil {
		return nil, err
	}
	if builder.replayAPI, err = replayAPI.NewHandlers(log, responser, sessionService, files); err != nil {
		return nil, err
	}

	// Initialize no-op assist proxy for session API handlers
	// Note: Assist (live sessions) feature is not enabled in HTTP service
	assistProxy := NewNoOpAssist()

	// Create adapter config for session API handlers
	sessionCfg := newSessionConfigAdapter(cfg)

	// Initialize session API handlers (provides /sessions/{id}/replay endpoint)
	if builder.sessionAPI, err = sessionAPI.NewHandlers(log, sessionCfg, responser, sessionService, assistProxy, files); err != nil {
		return nil, err
	}

	// Initialize events service (without ClickHouse - will return empty events)
	eventsService, err := events.New(log, nil)
	if err != nil {
		return nil, err
	}

	// Initialize event API handlers (provides /sessions/{id}/events endpoint)
	if builder.eventAPI, err = eventAPI.NewHandlers(log, &cfg.HTTP, responser, eventsService, sessionService); err != nil {
		return nil, err
	}

	return builder, nil
}
