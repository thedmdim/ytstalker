FROM golang:1.21 as builder
WORKDIR /build
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -v -o /app  .

FROM alpine
COPY --from=builder app /bin/app
EXPOSE 80
CMD ["/bin/app"]