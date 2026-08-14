# Sonix: document scanner and analysis app (Go API + React SPA).
# Multi-stage build: frontend -> Go binary -> minimal Alpine runtime.
# Base images are pinned by digest for reproducible, auditable builds.
# Stage 1: build frontend
FROM docker.io/library/node:20-alpine@sha256:09e2b3d9726018aecf269bd35325f46bf75046a643a66d28360ec71132750ec8 AS web
WORKDIR /app
COPY web/ ./
# Skip tsc so stub .d.ts don't conflict with @types/react in container; Vite still bundles.
RUN npm install && npx vite build

# Stage 2: build Go binary and copy static
FROM docker.io/library/golang:1.24-alpine@sha256:8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191 AS go
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
# Copy built frontend into embed path
COPY --from=web /app/dist ./cmd/server/static/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /sonix ./cmd/server

# Stage 3: minimal runtime (Tesseract OCR; eng+deu+osd language data from community)
FROM docker.io/library/alpine:3.19@sha256:6baf43584bcb78f2e5847d1de515f23499913ac9f12bdf834811a3145eb11ca1
RUN apk add --no-cache ca-certificates poppler-utils \
	&& apk add --no-cache --repository https://dl-cdn.alpinelinux.org/alpine/v3.19/community \
		tesseract-ocr tesseract-ocr-data-eng tesseract-ocr-data-deu tesseract-ocr-data-osd
WORKDIR /app
COPY --from=go /sonix /app/sonix
EXPOSE 9080 9443
ENV SERVER_ADDR=:9080 HTTPS_ADDR=:9443 DATA_DIR=/app/data \
	OCR_LANG=deu+eng OCR_DPI=300 OCR_PSM=1
VOLUME /app/data
ENTRYPOINT ["/app/sonix"]
