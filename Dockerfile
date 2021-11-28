FROM golang:1.17 as builder
WORKDIR /src
COPY . .
RUN go mod vendor
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GO111MODULE=on GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -mod=vendor -tags=netgo -o config-reloader cmd/main.go 


FROM alpine:3.15 AS final
WORKDIR /home/prometheus-for-ecs
COPY --from=builder /src/config-reloader .
ENV GO111MODULE=on
ENTRYPOINT ["./config-reloader"]
