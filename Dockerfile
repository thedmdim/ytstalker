FROM golang:1.20 as builder
WORKDIR /usr/src/app
COPY . .
RUN go mod download && go mod verify
RUN go build -v -o /usr/bin/app .

FROM busybox:1.35.0-uclibc as busybox
FROM gcr.io/distroless/static-debian11
COPY --from=busybox /bin/sh /bin/sh
WORKDIR /usr/bin/app
COPY --from=builder /usr/bin/app .
EXPOSE 80
ENTRYPOINT app