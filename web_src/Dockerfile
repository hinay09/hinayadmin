# ---- 构建阶段 ----
FROM node:22-alpine AS builder

RUN corepack enable && corepack prepare yarn@1.22.22 --activate

WORKDIR /build

COPY package.json yarn.lock ./
RUN yarn install --frozen-lockfile --ignore-scripts

COPY . .
RUN yarn install --frozen-lockfile
RUN yarn build

# ---- 运行阶段 ----
FROM node:22-alpine

RUN apk add --no-cache nginx

WORKDIR /app

COPY --from=builder /build/.output ./.output
COPY nginx.conf /etc/nginx/http.d/default.conf
COPY start.sh ./start.sh
RUN chmod +x ./start.sh

EXPOSE 80

CMD ["./start.sh"]
