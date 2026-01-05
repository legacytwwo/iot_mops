FROM golang:1.25.4 AS build
WORKDIR /src

COPY rule_engine/go.mod rule_engine/go.sum ./rule_engine/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd rule_engine && go mod download

COPY rule_engine/ ./rule_engine/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd rule_engine && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/rule-engine ./cmd/rule-engine

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/rule-engine /app/rule-engine
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/rule-engine"]
