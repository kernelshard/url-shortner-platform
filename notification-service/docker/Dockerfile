FROM golang:1.26.2-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
RUN go mod tidy

COPY . .

RUN go build -o server cmd/server/main.go

CMD ["./server"]
