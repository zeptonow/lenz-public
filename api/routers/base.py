from fastapi import APIRouter, Depends

from auth.auth_apikey import APIKeyAuth
from auth.auth_interservice import InterserviceAPIKeyAuth
from auth.auth_jwt import JWTAuth
from auth.auth_project import ProjectAuthorizer
from or_dependencies import ORRoute


def get_routers(prefix="", extra_dependencies=None, tags=None, include_internal: bool = False):
    if extra_dependencies is None:
        extra_dependencies = []
    if tags is None:
        tags = []

    public_app = APIRouter(route_class=ORRoute, prefix=prefix, tags=tags)
    app = APIRouter(
        dependencies=[Depends(JWTAuth()), Depends(ProjectAuthorizer("projectId"))] + extra_dependencies,
        route_class=ORRoute,
        prefix=prefix,
        tags=tags,
    )
    app_apikey = APIRouter(
        dependencies=[Depends(APIKeyAuth()), Depends(ProjectAuthorizer("projectKey"))] + extra_dependencies,
        route_class=ORRoute,
        prefix=prefix,
        tags=tags,
    )

    if not include_internal:
        return public_app, app, app_apikey

    internal_prefix = f"{prefix}/internal" if prefix else "/internal"
    app_internal = APIRouter(
        dependencies=[Depends(InterserviceAPIKeyAuth())] + extra_dependencies,
        route_class=ORRoute,
        prefix=internal_prefix,
        tags=tags,
    )
    return public_app, app, app_apikey, app_internal
