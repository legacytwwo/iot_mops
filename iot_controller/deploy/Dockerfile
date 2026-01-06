FROM golang:1.25.4 AS build
WORKDIR /src

COPY iot_controller/go.mod iot_controller/go.sum ./iot_controller/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd iot_controller && go mod download

COPY iot_controller/ ./iot_controller/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd iot_controller && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/iot-controller ./

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/iot-controller /app/iot-controller
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/iot-controller"]
