FROM golang:1.25-alpine AS builder

ARG SERVICE_CMD

WORKDIR /src

RUN apk add --no-cache ca-certificates git tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN if [ -z "${SERVICE_CMD}" ]; then echo "SERVICE_CMD build arg is required" && exit 1; fi

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/app \
    ${SERVICE_CMD}


FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/app /app/app

ENTRYPOINT ["/app/app"]