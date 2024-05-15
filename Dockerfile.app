FROM golang:1.21 as builder
WORKDIR /usr/src/ytstalker
COPY go.mod go.sum .
RUN go mod download
COPY cmd cmd/
RUN CGO_ENABLED=0 go build -v -o /usr/bin/ytstalker/app ./cmd/app

FROM gcr.io/distroless/static-debian11
WORKDIR /usr/bin/ytstalker
COPY web /usr/bin/ytstalker/web/
COPY --from=builder /usr/bin/ytstalker/app .
EXPOSE 80
ENTRYPOINT ["/usr/bin/ytstalker/app"]