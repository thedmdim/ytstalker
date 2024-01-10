FROM golang:1.21 as builder
WORKDIR /usr/src/app
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -v -o /usr/bin/app .

FROM alpine
WORKDIR /usr/bin/app
COPY --from=builder /usr/bin/app .
COPY frontend .
EXPOSE 80
CMD ["/usr/bin/app"]