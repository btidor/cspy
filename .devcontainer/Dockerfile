FROM mcr.microsoft.com/vscode/devcontainers/base:0-focal

ENV DEBIAN_FRONTEND=noninteractive
RUN apt update -q && apt upgrade -yq

# Install go
RUN (curl -sL https://go.dev/dl/go1.18.1.linux-amd64.tar.gz | tar xzC /usr/local) && \
    (echo 'PATH=$PATH:/usr/local/go/bin' >> /home/vscode/.profile)

# Install flyctl
ENV FLYCTL_INSTALL=/usr/local
RUN (curl https://fly.io/install.sh | sh) && rm /tmp/flyctl.*
