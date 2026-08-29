FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY apps ./apps
COPY database ./database
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/identity ./apps/identity/cmd/api

FROM alpine:3.21
RUN addgroup -S identity && adduser -S -G identity identity
COPY --from=build /out/identity /identity
COPY --from=build /src/database/identity/migrations /database/identity/migrations
USER identity
EXPOSE 8080
ENTRYPOINT ["/identity"]
