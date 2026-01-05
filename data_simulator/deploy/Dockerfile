FROM golang:1.25.4 AS build
WORKDIR /src

COPY data_simulator/go.mod data_simulator/go.sum ./data_simulator/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd data_simulator && go mod download

COPY data_simulator/ ./data_simulator/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd data_simulator && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/data-simulator ./

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/data-simulator /app/data-simulator
USER nonroot:nonroot
ENTRYPOINT ["/app/data-simulator"]
