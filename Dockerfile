# Stage 1: Build the WASM client
FROM golang:1.22-alpine AS wasm-builder
WORKDIR /build/client
COPY client/go.mod client/go.sum ./
RUN go mod download
COPY client/ .
RUN GOOS=js GOARCH=wasm go build -o /build/game.wasm .
RUN cp $(go env GOROOT)/lib/wasm/wasm_exec.js /build/wasm_exec.js 2>/dev/null || \
    cp $(go env GOROOT)/misc/wasm/wasm_exec.js /build/wasm_exec.js

# Stage 2: Build the Go server
FROM golang:1.22-alpine AS server-builder
WORKDIR /build/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
RUN go build -o /build/server ./cmd/server

# Stage 3: Minimal runtime image
FROM alpine:3.19
WORKDIR /app

COPY --from=server-builder /build/server ./server
COPY --from=wasm-builder   /build/game.wasm  ./static/game.wasm
COPY --from=wasm-builder   /build/wasm_exec.js ./static/wasm_exec.js
COPY static/ ./static/

EXPOSE 8080
ENTRYPOINT ["./server"]
