import hmac
import logging
from typing import Optional

from fastapi import Request
from fastapi.security import APIKeyHeader
from starlette import status
from starlette.exceptions import HTTPException

from or_config import config

logger = logging.getLogger(__name__)


class InterserviceAPIKeyAuth(APIKeyHeader):
    def __init__(self, expected: Optional[str] = None, auto_error: bool = True):
        super(InterserviceAPIKeyAuth, self).__init__(name="X-Api-Key", auto_error=auto_error)
        self._expected = expected

    def _expected_key(self) -> str:
        if self._expected is not None:
            return self._expected
        return (
            config("PY_INTERSERVICE_API_KEY", default="")
            or config("pythonInterserviceApiKey", default="")
        )

    async def __call__(self, request: Request):
        api_key: Optional[str] = await super(InterserviceAPIKeyAuth, self).__call__(request)
        expected = self._expected_key()
        if not expected or not api_key or not hmac.compare_digest(str(api_key), str(expected)):
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid interservice API key",
            )
        request.state.authorizer_identity = "interservice"
        return None

