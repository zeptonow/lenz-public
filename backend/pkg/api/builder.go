package api

import (
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"

	config "openreplay/backend/internal/config/api"
	"openreplay/backend/pkg/api_key"
	"openreplay/backend/pkg/canvas"
	"openreplay/backend/pkg/conditions"
	conditionsApi "openreplay/backend/pkg/conditions/projects_api"
	"openreplay/backend/pkg/db/postgres/pool"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/metrics/web"
	"openreplay/backend/pkg/objectstorage"
	"openreplay/backend/pkg/projects"
	replayAPI "openreplay/backend/pkg/replays/api"
	"openreplay/backend/pkg/replays/service"
	"openreplay/backend/pkg/server/api"
	"openreplay/backend/pkg/session"
	sessionAPI "openreplay/backend/pkg/session/api"
	"openreplay/backend/pkg/views"
)

type serviceBuilder struct {
	// Replay-only build: keep only ingestion/replay related handlers wired.
	sessionAPI    api.Handlers
	replayAPI     api.Handlers
	apiKeyAPI     api.Handlers
	conditionsAPI api.Handlers

	// Replay-only build: disabled products (kept for later).
	// eventAPI           api.Handlers
	// analyticsEventsAPI api.Handlers
	// favoriteAPI        api.Handlers
	// noteAPI            api.Handlers
	// cardsAPI           api.Handlers
	// dashboardsAPI      api.Handlers
	// chartsAPI          api.Handlers
	// searchAPI          api.Handlers
	// savedSearchesAPI   api.Handlers
	// usersAPI           api.Handlers
	// lexiconAPI         api.Handlers
}

func (b *serviceBuilder) Handlers() []api.Handlers {
	handlers := make([]api.Handlers, 0, 4)
	if b.sessionAPI != nil {
		handlers = append(handlers, b.sessionAPI)
	}
	if b.replayAPI != nil {
		handlers = append(handlers, b.replayAPI)
	}
	if b.apiKeyAPI != nil {
		handlers = append(handlers, b.apiKeyAPI)
	}
	if b.conditionsAPI != nil {
		handlers = append(handlers, b.conditionsAPI)
	}
	return handlers
}

func NewServiceBuilder(log logger.Logger, cfg *config.Config, webMetrics web.Web, pgconn pool.Pool, chconn clickhouse.Conn, objStore objectstorage.ObjectStorage, projects projects.Projects, canvases canvas.Canvases) (api.ServiceBuilder, error) {
	responser := api.NewResponser(webMetrics)

	// Replay-only build: analytics validation disabled (kept for later).
	// reqValidator := validator.New()
	// reqValidator.RegisterStructValidation(model.ValidateMetricFields, model.MetricPayload{})
	// reqValidator.RegisterStructValidation(model.ValidateFilterFields, model.Filter{})

	viewService, err := views.New(pgconn, chconn)
	if err != nil {
		return nil, fmt.Errorf("failed to create view service: %s", err)
	}

	// Replay-only build: assist disabled (kept for later).
	// assistProxy, err := proxy.New(log, cfg, projects)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create assist proxy: %s", err)
	// }

	files, err := service.New(log, cfg, objStore, canvases)
	if err != nil {
		return nil, err
	}

	sessionService, err := session.NewService(log, pgconn, viewService, files)
	if err != nil {
		return nil, fmt.Errorf("can't init session service: %s", err)
	}
	sessionHandlers, err := sessionAPI.NewHandlers(log, cfg, responser, sessionService, nil, files)
	if err != nil {
		return nil, err
	}

	replayHandlers, err := replayAPI.NewHandlers(log, responser, sessionService, files)
	if err != nil {
		return nil, err
	}

	apiKeyHandlers, err := api_key.NewHandlers(log, &cfg.HTTP, responser, projects)
	if err != nil {
		return nil, err
	}

	conditionsService := conditions.New(pgconn)
	conditionsHandlers, err := conditionsApi.NewHandlers(log, &cfg.HTTP, responser, conditionsService)
	if err != nil {
		return nil, err
	}

	return &serviceBuilder{
		sessionAPI:    sessionHandlers,
		replayAPI:     replayHandlers,
		apiKeyAPI:     apiKeyHandlers,
		conditionsAPI: conditionsHandlers,

		// Replay-only build: disabled products (kept for later).
		// eventAPI:           eventHandlers,
		// analyticsEventsAPI: analyticsEventsHandlers,
		// favoriteAPI:        favHandlers,
		// noteAPI:            noteHandlers,
		// cardsAPI:           cardsHandlers,
		// dashboardsAPI:      dashboardsHandlers,
		// chartsAPI:          chartsHandlers,
		// searchAPI:          searchHandlers,
		// savedSearchesAPI:   savedSearchesHandlers,
		// usersAPI:           usersHandlers,
		// lexiconAPI:         lexiconHandlers,
	}, nil
}
