# ---- build ----
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/surgalt ./cmd/server

# ---- runtime ----
FROM alpine:3.22
# ffmpeg: видео → WebM. LibreOffice: Word/PowerPoint → PDF (3D ном). libwebp: хурдан WebP кодлогч.
ARG WITH_OFFICE=1
RUN apk add --no-cache ca-certificates tzdata ffmpeg libwebp \
 && if [ "$WITH_OFFICE" = "1" ]; then apk add --no-cache libreoffice-writer libreoffice-impress libreoffice-calc font-noto; fi \
 && adduser -D -u 10001 app && mkdir -p /data && chown app /data
COPY --from=build /out/surgalt /usr/local/bin/surgalt
USER app
ENV ADDR=:8080 STORAGE_DIR=/data TZ=Asia/Ulaanbaatar
EXPOSE 8080 9090
HEALTHCHECK --interval=10s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["surgalt"]
