FROM golang:1.26.1-alpine
WORKDIR /app
COPY . .

RUN go build -o server ./cmd/server/main.go

CMD ["./server"]
