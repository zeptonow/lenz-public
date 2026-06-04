from or_config import config
from fastapi import Depends
from fastapi import HTTPException, status

import schemas
from chalicelib.core import health, tenants
from or_dependencies import OR_context
from routers.base import get_routers

public_app, app, app_apikey = get_routers()


@public_app.get('/healthz', tags=["health-check"])
async def get_global_health_status():
    return {"data": health.get_health()}


# Health check endpoint should always be available (for both public and authenticated)
@public_app.get('/health', tags=["health-check"])
async def get_public_health_status():
    # Return basic health check
    return {"data": health.get_health()}


@app.get('/health', tags=["health-check"])
def get_authenticated_health_status(context: schemas.CurrentContext = Depends(OR_context)):
    return {"data": health.get_health(context.tenant_id)}
