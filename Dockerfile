FROM golang:1.20 as builder
WORKDIR /usr/src/app
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -v -o /usr/bin/app .

FROM gcr.io/distroless/static-debian11
WORKDIR /usr/bin
COPY --from=builder /usr/bin/app .
EXPOSE 80
CMD ["/usr/bin/app"]