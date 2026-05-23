FROM node:22-alpine

ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0

WORKDIR /app

RUN npm install -g pnpm@10.19.0 --registry=https://registry.npmmirror.com
RUN pnpm config set registry https://registry.npmmirror.com

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .npmrc ./
COPY dev_core/package.json ./dev_core/package.json
RUN pnpm install --filter dev_core --prod --frozen-lockfile

COPY dev_core ./dev_core

EXPOSE 18101

WORKDIR /app/dev_core

CMD ["node", "src/index.js"]
