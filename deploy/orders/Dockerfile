FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY apps ./apps
COPY contracts ./contracts
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/orders ./apps/orders/cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && addgroup -S orders && adduser -S -G orders orders
COPY --from=build /out/orders /orders
COPY --from=build /src/contracts /contracts
USER orders
EXPOSE 8080
ENTRYPOINT ["/orders"]
