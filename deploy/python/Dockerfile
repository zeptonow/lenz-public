# syntax=docker/dockerfile:1

FROM registry.zepto.co.in/hardened-images/multi-arch/harden-python_3.12 AS builder
ARG ACCESS_TOKEN
USER root
RUN git config --global url."https://${ACCESS_TOKEN}:@github.com/zeptonow".insteadOf "https://github.com/zeptonow"

WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends build-essential && rm -rf /var/lib/apt/lists/*

COPY ./api/requirements.txt .
RUN python3 -m venv /venv && \
    /venv/bin/pip install --no-cache-dir --upgrade pip && \
    /venv/bin/pip install --no-cache-dir -r requirements.txt

FROM registry.zepto.co.in/hardened-images/multi-arch/harden-python_3.12
ARG GIT_SHA
LABEL GIT_SHA=$GIT_SHA

WORKDIR /app
USER root
ENV TZ=UTC \
    PYTHONUNBUFFERED=1 \
    PATH="/venv/bin:$PATH"

COPY --from=builder /venv /venv
COPY ./api .
RUN mv env.default .env

# Note: When VAULT_ENABLED=true in K8s, the app will read from /vault/secrets/static.json and /vault/secrets/dynamic.json
# When VAULT_ENABLED=false (local), it will use .env file
CMD ["/venv/bin/uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8000", "--proxy-headers", "--log-level", "warning"]
