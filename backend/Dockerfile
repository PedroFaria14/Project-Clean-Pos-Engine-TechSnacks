FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/create_user ./cmd/create_user \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/healthcheck ./cmd/healthcheck \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/dbcheck ./cmd/dbcheck

FROM alpine:3.24 AS runtime
ARG VCS_REF=unknown
LABEL org.opencontainers.image.revision=$VCS_REF
RUN apk add --no-cache ca-certificates \
    && addgroup -g 10001 app && adduser -D -H -u 10001 -G app app
COPY --from=build /out/ /usr/local/bin/
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
    CMD ["/usr/local/bin/healthcheck"]
CMD ["/usr/local/bin/api"]
