import logging
import time
from contextlib import asynccontextmanager

import psycopg_pool
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from or_config import config
from fastapi import FastAPI, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.gzip import GZipMiddleware
from psycopg import AsyncConnection
from psycopg.rows import dict_row
from starlette.responses import StreamingResponse

from chalicelib.utils import helper
from chalicelib.utils import pg_client, ch_client
from chalicelib.utils.security_logging import JWTRedactionFilter
from crons import core_crons, core_dynamic_crons
from routers import core, core_dynamic
from routers.subs import v1_api, health
# Replay-only build: disable analytics/integrations/spot/usability routers (kept for later).
# from routers.subs import insights, metrics, usability_tests, spot, product_analytics

loglevel = config("LOGLEVEL", default=logging.WARNING)
print(f">Loglevel set to: {loglevel}")
logging.basicConfig(level=loglevel)
logger = logging.getLogger(__name__)

# Apply JWT redaction filter to all loggers
jwt_redaction_filter = JWTRedactionFilter()
for handler in logging.root.handlers:
    handler.addFilter(jwt_redaction_filter)

# Also apply to this module's logger
for handler in logger.handlers:
    handler.addFilter(jwt_redaction_filter)


class ORPYAsyncConnection(AsyncConnection):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, row_factory=dict_row, **kwargs)


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    logging.info(">>>>> starting up <<<<<")
    ap_logger = logging.getLogger("apscheduler")
    ap_logger.setLevel(loglevel)

    app.schedule = AsyncIOScheduler()
    await pg_client.init()
    # Replay-only build: ClickHouse disabled (kept for later).
    # await ch_client.init()
    app.schedule.start()

    for job in core_crons.cron_jobs + core_dynamic_crons.cron_jobs:
        app.schedule.add_job(id=job["func"].__name__, **job)

    ap_logger.info(">Scheduled jobs:")
    for job in app.schedule.get_jobs():
        ap_logger.info(
            {
                "Name": str(job.id),
                "Run Frequency": str(job.trigger),
                "Next Run": str(job.next_run_time),
            }
        )

    database = {
        "host": config("pg_host", default="localhost"),
        "dbname": config("pg_dbname", default="orpy"),
        "user": config("pg_user", default="orpy"),
        "password": config("pg_password", default="orpy"),
        "port": config("pg_port", cast=int, default=5432),
        "application_name": "AIO" + config("APP_NAME", default="PY"),
    }

    database = psycopg_pool.AsyncConnectionPool(
        kwargs=database,
        connection_class=ORPYAsyncConnection,
        min_size=config("PG_AIO_MINCONN", cast=int, default=1),
        max_size=config("PG_AIO_MAXCONN", cast=int, default=5),
    )
    await database.open()
    app.state.postgresql = database

    # App listening
    yield

    # Shutdown
    await database.close()
    logging.info(">>>>> shutting down <<<<<")
    app.schedule.shutdown(wait=True)
    await pg_client.terminate()
    # await ch_client.terminate()


app = FastAPI(
    root_path=config("root_path", default="/api"),
    docs_url=config("docs_url", default=""),
    redoc_url=config("redoc_url", default=""),
    lifespan=lifespan,
)

# CORS middleware must be added first (runs last in the chain)
# This ensures preflight OPTIONS requests are handled correctly
# Note: Cannot use "*" with allow_credentials=True - browsers will reject it
# Add all frontend origins explicitly
origins = [
    "http://localhost:3333",
    "http://localhost:3000",
    "http://127.0.0.1:3333",
    "http://127.0.0.1:3000",
]

app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
    expose_headers=["*"],
)

app.add_middleware(GZipMiddleware, minimum_size=1000)

IGNORE_ENDPOINT_LOG = ["/"]


@app.middleware("http")
async def log_all_requests(request: Request, call_next):
    method = request.method
    endpoint = request.url.path
    response: Response = await call_next(request)
    # Log all endpoints except health check
    if endpoint not in IGNORE_ENDPOINT_LOG or response.status_code != 200:
        logger.info(f"{method}:{endpoint} {response.status_code}")
    return response


@app.middleware("http")
async def or_middleware(request: Request, call_next):
    if helper.TRACK_TIME:
        now = time.time()
    try:
        response: StreamingResponse = await call_next(request)
    except Exception:
        logger.exception(f"{request.method}: {request.url.path} FAILED!")
        raise
    if response.status_code // 100 != 2:
        logging.warning(f"{request.method}:{request.url.path} {response.status_code}!")
    if helper.TRACK_TIME:
        now = time.time() - now
        if now > 2:
            now = round(now, 2)
            logging.warning(f"Execution time: {now} s for {request.method}: {request.url.path}")
    response.headers["x-robots-tag"] = 'noindex, nofollow'
    response.headers["X-Content-Type-Options"] = "nosniff"
    response.headers["X-Frame-Options"] = "SAMEORIGIN"
    response.headers["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains"
    return response
app.include_router(core.public_app)
app.include_router(core.app)
app.include_router(core.app_apikey)
app.include_router(core_dynamic.public_app)
app.include_router(core_dynamic.app)
app.include_router(core_dynamic.app_apikey)
app.include_router(v1_api.app_apikey)
app.include_router(health.public_app)
app.include_router(health.app)
app.include_router(health.app_apikey)

# Replay-only build: keep router mounts commented for later.
# app.include_router(metrics.app)
# app.include_router(insights.app)
#
# app.include_router(usability_tests.public_app)
# app.include_router(usability_tests.app)
# app.include_router(usability_tests.app_apikey)
#
# app.include_router(spot.public_app)
# app.include_router(spot.app)
# app.include_router(spot.app_apikey)
#
# app.include_router(product_analytics.public_app, prefix="/pa")
# app.include_router(product_analytics.app, prefix="/pa")
# app.include_router(product_analytics.app_apikey, prefix="/pa")
