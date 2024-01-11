FROM golang:1.21 as builder
WORKDIR /usr/src/ytstalker
COPY go.mod .
RUN go mod download
COPY app app/
RUN ls
RUN CGO_ENABLED=0 go build -v -o /usr/bin/ytstalker/app .

FROM alpine
WORKDIR /usr/bin/ytstalker
COPY --from=builder /usr/bin/ytstalker/app .
COPY web /usr/bin/ytstalker/web
EXPOSE 80
ENTRYPOINT ["/usr/bin/ytstalker/app"]