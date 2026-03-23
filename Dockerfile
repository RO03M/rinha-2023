FROM golang:1.24.4-alpine AS builder

WORKDIR /app

COPY go.mod ./
# COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /rinha

FROM scratch

COPY --from=builder /rinha /rinha

EXPOSE 80

CMD ["/rinha"]