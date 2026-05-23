FROM node:22-alpine

ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0

WORKDIR /app

RUN npm install -g pnpm@10.19.0 --registry=https://registry.npmmirror.com
RUN pnpm config set registry https://registry.npmmirror.com

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --prod --frozen-lockfile

COPY . .

EXPOSE 19601

CMD ["node", "src/index.js"]
