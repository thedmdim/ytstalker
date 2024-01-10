FROM golang:1.21 as builder
WORKDIR /build
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -v -o /app  .

FROM alpine
COPY --from=builder /app /app
EXPOSE 80
CMD ["/app"]