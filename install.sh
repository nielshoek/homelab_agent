#!/bin/bash

SERVICE_NAME="homelab_agent"
SERVICE_PATH="/etc/systemd/system/${SERVICE_NAME}.service"
INSTALL_DIR="/usr/local/bin/${SERVICE_NAME}"
REPO_URL="https://api.github.com/repos/nielshoek/homelab_agent"

if [ -z "$DEPLOY_TOKEN" ] || [ -z "GITHUB_TOKEN" ]; then
  echo "Error: DEPLOY_TOKEN and GITHUB_TOKEN must be set as environment variables."
  exit 1
fi

PORT=${PORT:-9090}

# Determine system architecture
ARCH=$(uname -m)
if [ "$ARCH" == "x86_64" ]; then
  ARCH="amd64"
elif [ "$ARCH" == "aarch64" ]; then
  ARCH="arm64"
else
  echo "Error: Unsupported architecture $ARCH"
  exit 1
fi

# Fetch the latest release information
LATEST_RELEASE=$(curl -sL "$REPO_URL/releases/latest")
if [ -z "$LATEST_RELEASE" ]; then
  echo "Error: Unable to fetch the latest release."
  exit 1
fi

# Extract the download URL for the appropriate binary
DOWNLOAD_URL=$(echo "$LATEST_RELEASE" | grep -oP "https://github.com/nielshoek/homelab_agent/releases/download/.*/homelab_agent_${ARCH}" | head -n 1)
if [ -z "$DOWNLOAD_URL" ]; then
  echo "Error: Unable to find a binary for architecture $ARCH."
  exit 1
fi

# Check if service is already installed
SERVICE_INSTALLED=false
if systemctl is-active --quiet $SERVICE_NAME 2>/dev/null || systemctl is-enabled --quiet $SERVICE_NAME 2>/dev/null; then
  SERVICE_INSTALLED=true
  echo "Existing installation detected. Updating homelab_agent..."
  sudo systemctl stop $SERVICE_NAME
fi

# Download the binary
curl -sL -o homelab_agent "$DOWNLOAD_URL"
chmod +x homelab_agent

sudo mkdir -p "$INSTALL_DIR"

sudo mv ./homelab_agent "$INSTALL_DIR"

sudo bash -c "cat > $SERVICE_PATH" <<EOL
[Unit]
Description=Homelab Agent Service
After=network.target

[Service]
ExecStart=${INSTALL_DIR}/homelab_agent
WorkingDirectory=${INSTALL_DIR}
Restart=always
Environment="DEPLOY_TOKEN=${DEPLOY_TOKEN}"
Environment="GITHUB_TOKEN=${GITHUB_TOKEN}"
Environment="PORT=${PORT}"
User=nielshoek
Group=nielshoek

[Install]
WantedBy=multi-user.target
EOL

sudo systemctl daemon-reload

if [ "$SERVICE_INSTALLED" = true ]; then
  sudo systemctl start $SERVICE_NAME
  echo "Homelab Agent updated and restarted successfully."
else
  sudo systemctl enable "$SERVICE_NAME"
  sudo systemctl start "$SERVICE_NAME"
  echo "Homelab Agent installed and started successfully."
fi

echo "Run 'sudo journalctl -u $SERVICE_NAME | tail' to check the logs."