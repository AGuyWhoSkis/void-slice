# syntax=docker/dockerfile:1.7

FROM golang:1.23-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w" \
    -o /out/voidslice ./cmd/voidslice

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/voidslice /voidslice
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/voidslice"]
CMD ["serve", "--port", "8080"]
