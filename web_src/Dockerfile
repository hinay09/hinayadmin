# ---- 构建阶段 ----
FROM node:22-alpine AS builder

# 国内加速: corepack 下载 yarn 本体与依赖安装均走 npmmirror
ENV COREPACK_NPM_REGISTRY=https://registry.npmmirror.com

RUN corepack enable && corepack prepare yarn@1.22.22 --activate \
    && yarn config set registry https://registry.npmmirror.com

WORKDIR /build

COPY package.json ./
RUN yarn install --ignore-scripts

COPY . .
RUN yarn install
RUN yarn build

# ---- 运行阶段 ----
FROM node:22-alpine

RUN sed -i 's#https://dl-cdn.alpinelinux.org#https://mirrors.aliyun.com#g' /etc/apk/repositories \
    && apk add --no-cache nginx

WORKDIR /app

COPY --from=builder /build/.output ./.output
COPY nginx.conf /etc/nginx/http.d/default.conf
COPY start.sh ./start.sh
RUN chmod +x ./start.sh

EXPOSE 80

CMD ["./start.sh"]
