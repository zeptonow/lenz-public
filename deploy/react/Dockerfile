# syntax=docker/dockerfile:1

FROM registry_url AS builder
ARG NODE_OPTIONS="--max-old-space-size=10240"
ENV NODE_OPTIONS=$NODE_OPTIONS
USER root

WORKDIR /work
COPY ./frontend .
RUN cp .env.sample .env
RUN yarn && yarn build

FROM registry_url
ARG GIT_SHA
LABEL GIT_SHA=$GIT_SHA
USER root

COPY --from=builder /work/public /var/www/openreplay
COPY ./frontend/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
