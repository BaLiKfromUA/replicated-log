FROM golang:latest as builder

RUN mkdir /app
ADD . /app/
WORKDIR /app

RUN go mod tidy
RUN go build -o main ./cmd/

FROM alpine:3.14
COPY --from=builder /app/main .
CMD ["./main"]