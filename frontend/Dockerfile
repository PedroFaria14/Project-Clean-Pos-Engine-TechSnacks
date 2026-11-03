FROM node:24.18.0-alpine AS dependencies
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM dependencies AS development
COPY . .
RUN chown -R node:node /app
USER node
EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]

FROM dependencies AS build
COPY . .
RUN npm run build

FROM nginxinc/nginx-unprivileged:1.31-alpine AS runtime
ARG VCS_REF=unknown
LABEL org.opencontainers.image.revision=$VCS_REF
USER root
RUN apk add --no-cache ca-certificates
COPY nginx.conf /etc/nginx/techsnacks.template
COPY --chmod=0555 docker/start_frontend.sh /usr/local/bin/start_frontend
COPY --from=build /app/dist/ /usr/share/nginx/html/
USER 101:101
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${PORT:-8080}/health/live" || exit 1
ENTRYPOINT ["/usr/local/bin/start_frontend"]
CMD ["nginx", "-c", "/tmp/techsnacks-nginx.conf", "-g", "daemon off;"]
