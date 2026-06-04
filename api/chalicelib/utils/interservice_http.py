from __future__ import annotations

from typing import Any, Dict, Literal, Optional

import requests

from or_config import config


HEADER_NAME = "X-Api-Key"
TargetStack = Literal["go", "python"]


def _read_key(*, target: TargetStack) -> str:
    """
    Reads the interservice key from Vault/env.

    Supports both the plan's env-style names (GO_INTERSERVICE_API_KEY / PY_INTERSERVICE_API_KEY)
    and camelCase names (goInterserviceApiKey / pythonInterserviceApiKey) for compatibility with
    Go's Viper/mapstructure config style.
    """
    if target == "go":
        return (
            config("GO_INTERSERVICE_API_KEY", default="")
            or config("goInterserviceApiKey", default="")
        )
    return (
        config("PY_INTERSERVICE_API_KEY", default="")
        or config("pythonInterserviceApiKey", default="")
    )


def headers_for_target(target: TargetStack, extra: Optional[Dict[str, str]] = None) -> Dict[str, str]:
    headers: Dict[str, str] = {}
    if extra:
        headers.update(extra)
    key = _read_key(target=target)
    if key:
        headers[HEADER_NAME] = key
    return headers


def request_to_target(
    target: TargetStack,
    method: str,
    url: str,
    *,
    headers: Optional[Dict[str, str]] = None,
    **kwargs: Any,
) -> requests.Response:
    merged = headers_for_target(target, headers)
    return requests.request(method=method, url=url, headers=merged, **kwargs)


def get_to_target(target: TargetStack, url: str, *, headers: Optional[Dict[str, str]] = None, **kwargs: Any) -> requests.Response:
    return request_to_target(target, "GET", url, headers=headers, **kwargs)


def post_to_target(target: TargetStack, url: str, *, headers: Optional[Dict[str, str]] = None, **kwargs: Any) -> requests.Response:
    return request_to_target(target, "POST", url, headers=headers, **kwargs)

