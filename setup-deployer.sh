#!/bin/bash
# Deployer User Setup Script
# Run this on the Ubuntu 22.04 server

set -e

USERNAME="deployer"
USER_HOME="/home/$USERNAME"

# Create user if doesn't exist
if ! id "$USERNAME" &>/dev/null; then
    echo "Creating user: $USERNAME"
    sudo useradd -m -s /bin/bash "$USERNAME"
else
    echo "User $USERNAME already exists"
fi

# Add to sudoers for deployment tasks
if ! sudo grep -q "$USERNAME" /etc/sudoers; then
    echo "Adding $USERNAME to sudoers (no-password for docker, systemctl)"
    echo "$USERNAME ALL=(ALL) NOPASSWD: /usr/bin/docker, /usr/bin/systemctl, /usr/bin/apt, /usr/bin/make, /usr/local/bin/docker-compose, /usr/bin/docker-compose" | sudo tee -a /etc/sudoers.d/deployer
    sudo chmod 440 /etc/sudoers.d/deployer
fi

# Add to docker group if docker exists
if getent group docker >/dev/null; then
    sudo usermod -aG docker "$USERNAME"
    echo "Added $USERNAME to docker group"
fi

# Setup SSH directory
sudo mkdir -p "$USER_HOME/.ssh"
sudo chmod 700 "$USER_HOME/.ssh"

# Add public key
cat > /tmp/deployer_key.pub << 'EOF'
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIWFiMtTW7c3r6ttU5GkEnjqEQuMpwVRT1ucXnvUlUHd deployer@hystersis
EOF

sudo cp /tmp/deployer_key.pub "$USER_HOME/.ssh/authorized_keys"
sudo chmod 600 "$USER_HOME/.ssh/authorized_keys"
sudo chown -R "$USERNAME:$USERNAME" "$USER_HOME/.ssh"
rm -f /tmp/deployer_key.pub

# Install essential deployment tools
echo "Installing deployment tools..."
sudo apt-get update -qq
sudo apt-get install -y -qq git curl make jq unzip 2>/dev/null || true

# Install Docker if not present
if ! command -v docker &>/dev/null; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com | sh
    sudo usermod -aG docker "$USERNAME"
fi

# Install Docker Compose
if ! command -v docker-compose &>/dev/null; then
    echo "Installing Docker Compose..."
    sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
fi

# Setup deployment directory
sudo mkdir -p "$USER_HOME/apps"
sudo chown "$USERNAME:$USERNAME" "$USER_HOME/apps"

# Setup git defaults
sudo -u "$USERNAME" git config --global init.defaultBranch main 2>/dev/null || true

echo ""
echo "========================================"
echo "✅ Deployer user setup complete!"
echo "========================================"
echo "Username: $USERNAME"
echo "Home: $USER_HOME"
echo "Apps dir: $USER_HOME/apps"
echo ""
echo "SSH Access:"
echo "  ssh -i deployer_key deployer@54.87.249.176"
echo ""
echo "Deployment workflow:"
echo "  1. SSH into server: ssh -i deployer_key deployer@54.87.249.176"
echo "  2. Go to apps: cd ~/apps"
echo "  3. Clone your project: git clone <your-repo>"
echo "  4. Deploy: cd <project> && docker-compose up -d"
echo "========================================"
