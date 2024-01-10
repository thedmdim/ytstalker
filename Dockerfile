FROM golang:1.21 as builder
WORKDIR /build
COPY go.mod .
RUN go mod download
COPY . .
RUN go build CGO_ENABLED=0 -v -o /app  .

FROM alpine
COPY --from=builder /app /app
EXPOSE 80
CMD ["/app"]