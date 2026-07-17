#!/bin/bash
# Password SSH User Setup Script
# Run this on the Ubuntu 22.04 server (hystersis-backend)

set -e

# Configuration
USERNAME="${1:-teamuser}"
PASSWORD="${2:-$(openssl rand -base64 12)}"

echo "========================================"
echo "🔐 Password SSH User Setup"
echo "========================================"
echo "Username: $USERNAME"
echo "Generated Password: $PASSWORD"
echo ""

# Create user
if ! id "$USERNAME" &>/dev/null; then
    echo "Creating user: $USERNAME"
    sudo useradd -m -s /bin/bash "$USERNAME"
else
    echo "User $USERNAME already exists"
fi

# Set password
echo "$USERNAME:$PASSWORD" | sudo chpasswd
echo "✅ Password set for $USERNAME"

# Ensure user can use sudo (optional - remove if not needed)
if ! sudo grep -q "$USERNAME" /etc/sudoers.d/ 2>/dev/null; then
    echo "Adding sudo access for $USERNAME"
    echo "$USERNAME ALL=(ALL) ALL" | sudo tee "/etc/sudoers.d/$USERNAME" >/dev/null
    sudo chmod 440 "/etc/sudoers.d/$USERNAME"
fi

# Enable password authentication in SSH if not already enabled
SSHD_CONFIG="/etc/ssh/sshd_config"
if sudo grep -q "^PasswordAuthentication no" "$SSHD_CONFIG" 2>/dev/null; then
    echo "🔧 Enabling SSH password authentication..."
    sudo sed -i 's/^PasswordAuthentication no/PasswordAuthentication yes/' "$SSHD_CONFIG"
    sudo systemctl restart sshd
    echo "✅ SSH password authentication enabled"
else
    echo "✅ SSH password authentication already enabled (or using default)"
fi

# Also ensure ChallengeResponseAuthentication is set properly
if sudo grep -q "^ChallengeResponseAuthentication no" "$SSHD_CONFIG" 2>/dev/null; then
    sudo sed -i 's/^ChallengeResponseAuthentication no/ChallengeResponseAuthentication yes/' "$SSHD_CONFIG"
    sudo systemctl restart sshd
fi

# Create apps directory
sudo mkdir -p "/home/$USERNAME/apps"
sudo chown "$USERNAME:$USERNAME" "/home/$USERNAME/apps"

echo ""
echo "========================================"
echo "✅ User setup complete!"
echo "========================================"
echo ""
echo "📋 Login Details:"
echo "  Username: $USERNAME"
echo "  Password: $PASSWORD"
echo "  Host:     54.87.249.176"
echo ""
echo "🔑 SSH Login:"
echo "  ssh $USERNAME@54.87.249.176"
echo ""
echo "⚠️  IMPORTANT:"
echo "  1. Change the password after first login:"
echo "     passwd"
echo ""
echo "  2. To create additional users, run:"
echo "     sudo bash /tmp/setup-password-user.sh <username> <password>"
echo ""
echo "========================================"
