FROM node:22-alpine AS builder

ARG WEB_APP_PATH=web/morningapp

ARG VITE_WAKE_PLAN_SERVICE=http
ARG VITE_API_BASE_URL=https://morning.veloranet.ru
ARG VITE_API_REQUEST_TIMEOUT_MS=10000

ENV VITE_WAKE_PLAN_SERVICE=${VITE_WAKE_PLAN_SERVICE}
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV VITE_API_REQUEST_TIMEOUT_MS=${VITE_API_REQUEST_TIMEOUT_MS}

WORKDIR /src

COPY ${WEB_APP_PATH}/package.json ${WEB_APP_PATH}/package-lock.json ./

RUN npm ci

COPY ${WEB_APP_PATH}/ ./

RUN npm run build


FROM nginx:1.27-alpine

COPY --from=builder /src/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /src/dist /usr/share/nginx/html

EXPOSE 8080