FROM golang:1.23.4 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server main.go

FROM ubuntu:latest
WORKDIR /app
COPY --from=builder /app/server .
COPY web ./web
EXPOSE 7540

ENV TODO_PORT=7540
ENV TODO_DBFILE=/app/scheduler.db
ENV TODO_PASSWORD=password

CMD ["./server"]