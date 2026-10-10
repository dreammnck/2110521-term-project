FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SERVICE
RUN CGO_ENABLED=0 go build -trimpath -o /out/app ./cmd/${SERVICE}

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D app && mkdir -p /data/photos && chown -R app /data
COPY --from=build /out/app /app
USER app
ENTRYPOINT ["/app"]
