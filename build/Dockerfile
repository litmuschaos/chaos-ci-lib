FROM golang:alpine AS builder

# Set Go environment variables
ENV GO111MODULE=off
ENV GOPATH=/go
ENV PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

# Install dependencies and cleanup
RUN apk update && \
    apk add --no-cache \
        git \
        wget \
        tar

# Install Helm
RUN wget https://get.helm.sh/helm-v3.4.0-linux-amd64.tar.gz && \
    tar -zxvf helm-v3.4.0-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    rm -rf helm-v3.4.0-linux-amd64.tar.gz linux-amd64

# Install kubectl
RUN wget https://storage.googleapis.com/kubernetes-release/release/v1.18.0/bin/linux/amd64/kubectl -O /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl

# Copy and build the binaries
COPY build/_output /app/_output
COPY litmus/helm-install.sh /app/
COPY build/experiment_entrypoint.sh /app/

FROM alpine:latest

# Install runtime dependencies
RUN apk update && \
    apk add --no-cache \
        openssh-client

# Copy only the necessary files from the build stage
COPY --from=builder /app /app
COPY --from=builder /usr/local/bin/kubectl /usr/local/bin/kubectl
COPY --from=builder /usr/local/bin/helm /usr/local/bin/helm

# Set working directory
WORKDIR /app

# Override entrypoint with test binary
ENTRYPOINT ["./all-experiments"]
