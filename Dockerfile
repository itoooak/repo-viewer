FROM golang:1.26.3 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ./bin/server .

FROM gcr.io/distroless/static-debian13 AS final

USER nonroot:nonroot
WORKDIR /app

COPY --from=builder /app/bin/server .
COPY --from=builder /app/templates ./templates

ENV REPO_DIR=/app/repos
ENV SERVER_PORT=8080

EXPOSE 8080
ENTRYPOINT ["./server"]
