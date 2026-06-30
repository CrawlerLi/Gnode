FROM golang:1.25 AS builder
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/gnode ./cmd

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /out/gnode /app/
COPY --from=builder /src/configs /app/configs
ENTRYPOINT [ "/app/gnode" ]