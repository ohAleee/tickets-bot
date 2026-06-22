# Dashboard frontend (Svelte SPA) — built from local source (the upstream Dockerfile
# cloned from GitHub; here we build the vendored ./frontend unchanged). Build-time ARGs
# are baked into the bundle by `npm run build`.
FROM node:lts-alpine AS build

RUN apk add --no-cache git
RUN mkdir -p /app && chown -R node:node /app
WORKDIR /app
USER node

COPY --chown=node:node package*.json ./
RUN npm install

COPY --chown=node:node . .

ARG CLIENT_ID
ARG REDIRECT_URI
ARG API_URL
ARG WS_URL
ARG FRONTPAGE_URL
ARG REDIRECT_WEBSITE
ARG DOCS_URL
ARG INVITE_URL
ARG TITLE
ARG DESCRIPTION
ARG FAVICON
ARG FAVICON_TYPE
ARG WHITELABEL_DISABLED

RUN npm run build
RUN npm prune --production

# Production container
FROM node:lts-alpine AS prod

RUN mkdir -p /app && chown -R node:node /app
WORKDIR /app
USER node

COPY --from=build --chown=node:node /app/package*.json /app/
COPY --from=build --chown=node:node /app/node_modules /app/node_modules
COPY --from=build --chown=node:node /app/public /app/public

ENV NODE_ENV=production
CMD ["npx", "sirv", "public", "-s", "--no-clear", "--host", "0.0.0.0", "--port", "5000"]
