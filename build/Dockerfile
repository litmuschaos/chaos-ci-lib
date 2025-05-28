FROM golang:1.24-alpine AS builder

LABEL maintainer="LitmusChaos"

# Install build dependencies
RUN apk add --no-cache git

# Set up Go environment for Linux cross-compilation
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build all the Go test binaries with proper Linux targeting
RUN mkdir -p build/_output

# Build individual experiment binaries
RUN go test -o build/_output/install-litmus -c litmus/install-litmus_test.go -v -count=1
RUN go test -o build/_output/uninstall-litmus -c litmus/uninstall-litmus_test.go -v -count=1
RUN go test -o build/_output/pod-delete -c experiments/pod-delete_test.go -v -count=1
RUN go test -o build/_output/container-kill -c experiments/container-kill_test.go -v -count=1
RUN go test -o build/_output/pod-cpu-hog -c experiments/pod-cpu-hog_test.go -v -count=1
RUN go test -o build/_output/pod-memory-hog -c experiments/pod-memory-hog_test.go -v -count=1
RUN go test -o build/_output/node-cpu-hog -c experiments/node-cpu-hog_test.go -v -count=1
RUN go test -o build/_output/node-memory-hog -c experiments/node-memory-hog_test.go -v -count=1
RUN go test -o build/_output/pod-network-corruption -c experiments/pod-network-corruption_test.go -v -count=1
RUN go test -o build/_output/pod-network-latency -c experiments/pod-network-latency_test.go -v -count=1
RUN go test -o build/_output/pod-network-loss -c experiments/pod-network-loss_test.go -v -count=1
RUN go test -o build/_output/disk-fill -c experiments/disk-fill_test.go -v -count=1

# Build the all-experiments binary
RUN cd experiments && go test -o ../build/_output/all-experiments -c -v -count=1

# Final runtime stage
FROM ubuntu:20.04

LABEL maintainer="LitmusChaos"

# Prevent timezone prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=UTC

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    git \
    ssh \
    rsync \
    curl \
    wget \
    ca-certificates \
    file \
    && rm -rf /var/lib/apt/lists/*

ARG KUBECTL_VERSION=1.18.0

# Installing helm
RUN wget https://get.helm.sh/helm-v3.4.0-linux-amd64.tar.gz && \ 
    tar -zxvf helm-v3.4.0-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    rm -rf helm-v3.4.0-linux-amd64.tar.gz linux-amd64

# Install kubectl
ADD https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

# Set working directory
WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder /app/build/_output ./
COPY litmus/helm-install.sh ./ 
COPY build/experiment_entrypoint.sh ./

# Make all binaries executable
RUN chmod +x ./*

# Verify the binaries are correct format
RUN file ./container-kill && echo "Binary format verified"
RUN file ./all-experiments && echo "All-experiments binary format verified"

# Set entrypoint
ENTRYPOINT ["./all-experiments"]
