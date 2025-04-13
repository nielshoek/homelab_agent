#!/bin/bash
DEPLOY_TOKEN="your-deploy-token" GITHUB_TOKEN="your-github-token" bash -c "$(curl -fsSL https://raw.githubusercontent.com/nielshoek/homelab_agent/main/install.sh || echo 'echo \"Error: Failed to download install script\" >&2; exit 1')"
