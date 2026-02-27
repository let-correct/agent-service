# ---- Build stage ----
FROM --platform=linux/arm64 golang:1.23-alpine AS builder

WORKDIR /app

# Download dependencies first (layer-cached unless go.mod/go.sum change)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a statically linked binary for Lambda
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -ldflags="-s -w" -o bootstrap ./cmd/google-calendar-auth

# ---- Runtime stage ----
# AWS provided base image for arm64 — includes the Lambda Runtime Interface Client
FROM --platform=linux/arm64 public.ecr.aws/lambda/provided:al2023-arm64

# Copy the compiled binary
COPY --from=builder /app/bootstrap /var/runtime/bootstrap

# Lambda expects the handler to be named "bootstrap" for custom runtimes
CMD ["bootstrap"]