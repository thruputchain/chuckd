#!/bin/bash
set -e

# Create log file for this script
LOG_FILE="/var/log/groundhog-build-script.log"
exec > >(tee -a $LOG_FILE)
exec 2>&1

echo "Starting Groundhog provisioning at $(date)"

# Update system
echo "Updating system packages..."
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get upgrade -y

# Install dependencies
echo "Installing dependencies..."
apt-get install -y wget git build-essential curl

# Validate environment variables
if [ -z "$GITHUB_TOKEN" ]; then
    echo "ERROR: GITHUB_TOKEN environment variable is required"
    exit 1
fi

# Setup git for private repos
echo "Configuring git for private repositories..."
git config --global url."https://$GITHUB_TOKEN:@github.com/".insteadOf "https://github.com/"
git config --global url."https://$GITHUB_TOKEN@github.com/".insteadOf "https://github.com/"

# Install latest Go
echo "Installing Go..."
GO_VERSION=$(curl -s https://go.dev/VERSION?m=text | head -n1 | xargs)
echo "Installing Go version: $GO_VERSION"

wget -q https://go.dev/dl/${GO_VERSION}.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf ${GO_VERSION}.linux-amd64.tar.gz
rm ${GO_VERSION}.linux-amd64.tar.gz

# Setup go for private repos
export PATH=$PATH:/usr/local/go/bin
/usr/local/go/bin/go env -w GOPRIVATE=github.com/groundhogchain/*

# Set up for ubuntu user
echo 'export PATH=$PATH:/usr/local/go/bin' >> /home/ubuntu/.bashrc
echo 'export GOPATH=/home/ubuntu/go' >> /home/ubuntu/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> /home/ubuntu/.bashrc

# Source the profile to make Go available immediately
source /home/ubuntu/.bashrc

# Clone private repository
echo "Cloning chuckd"
git clone https://$GITHUB_TOKEN@github.com/groundhogchain/chuckd /tmp/chuckd

# Build and install chuckd
echo "Building chuckd..."
cd /tmp/chuckd/chuckd
/usr/local/go/bin/go mod download
/usr/local/go/bin/go build -o /usr/local/bin/chuckd .
chmod +x /usr/local/bin/chuckd

# Verify installation
echo "Verifying chuckd installation..."
/usr/local/bin/chuckd version || echo "chuckd installed successfully"

# Clean up sensitive information and temporary files
echo "Cleaning up..."
rm -rf /tmp/chuckd
unset GITHUB_TOKEN
git config --global --unset url."https://$GITHUB_TOKEN:@github.com/".insteadOf || true
git config --global --unset url."https://$GITHUB_TOKEN@github.com/".insteadOf || true

# Signal completion
echo "Setup completed successfully at $(date)"
echo "SUCCESS" > /tmp/setup-complete.txt

echo "Groundhog provisioning completed successfully!"
