FROM node:22-alpine AS frontend-build

ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0

WORKDIR /repo

RUN npm install -g pnpm@10.19.0 --registry=https://registry.npmmirror.com
RUN pnpm config set registry https://registry.npmmirror.com
RUN pnpm config set fetch-retries 5
RUN pnpm config set fetch-retry-mintimeout 10000
RUN pnpm config set fetch-retry-maxtimeout 120000
RUN pnpm config set network-concurrency 4

COPY . .

RUN cd dev_ide && pnpm install --frozen-lockfile && pnpm build
RUN cd datacenter && pnpm install --frozen-lockfile && pnpm build
RUN cd designer && pnpm install --frozen-lockfile && pnpm build

FROM nginx:1.27-alpine

COPY scripts/docker/edge/nginx.conf /etc/nginx/nginx.conf
COPY --from=frontend-build /repo/dev_ide/dist /usr/share/nginx/html
COPY --from=frontend-build /repo/datacenter/dist /usr/share/nginx/html/datacenter
COPY --from=frontend-build /repo/designer/dist /usr/share/nginx/html/designer

EXPOSE 80
