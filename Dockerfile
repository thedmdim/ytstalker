FROM golang:1.20 as builder
WORKDIR /usr/src/app
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -v -o /usr/bin/app .

FROM alpine
COPY --from=builder /usr/bin/app /usr/bin/app
EXPOSE 80
CMD ["/usr/bin/app"]