FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /

COPY --from=builder /server /server

ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/server"]
