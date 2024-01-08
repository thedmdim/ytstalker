FROM golang:1.20 as builder
WORKDIR /usr/src/app
COPY go.mod .
RUN go mod download
COPY . .
RUN go mod download && go mod verify
RUN go build -v -o /usr/bin/app .

FROM gcr.io/distroless/static-debian11
WORKDIR /usr/bin/app
COPY --from=builder /usr/bin/app .
EXPOSE 80
CMD ["/usr/bin/app"]