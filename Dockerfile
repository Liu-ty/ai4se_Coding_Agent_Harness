# syntax=docker/dockerfile:1
FROM node:24-bookworm AS web-build
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26.5-bookworm AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=web-build /src/internal/httpapi/webdist ./internal/httpapi/webdist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/ai4se-harness ./cmd/ai4se-harness

FROM gcr.io/distroless/base-debian12:nonroot AS demo
COPY --from=go-build /out/ai4se-harness /usr/local/bin/ai4se-harness
COPY --from=go-build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/ai4se-harness", "serve", "--profile", "demo", "--addr", "0.0.0.0:8080"]
