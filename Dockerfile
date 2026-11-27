FROM golang:1.27.4-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /matchmaking-service ./cmd/server

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /matchmaking-service /matchmaking-service
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/matchmaking-service"]
