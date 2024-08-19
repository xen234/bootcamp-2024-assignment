FROM golang:alpine AS builder

WORKDIR /build

COPY . .

RUN apk update && \
    apk add --no-cache gcc musl-dev

ENV CGO_ENABLED=1

RUN go build -o main ./main.go

FROM alpine

WORKDIR /build

COPY --from=builder /build/main /build/main

CMD ["./main"]

EXPOSE 8081