FROM golang:latest

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY . .
RUN go mod tidy && go build -v -o app
ENTRYPOINT app
