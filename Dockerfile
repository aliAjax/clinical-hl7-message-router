FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/router ./cmd/router

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/router /router
EXPOSE 8084 2575
USER nonroot:nonroot
ENTRYPOINT ["/router"]
