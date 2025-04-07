# Digital Ocean Deployment Guide

This guide provides detailed instructions for setting up the Water Bot on a Digital Ocean Droplet with automated deployments using GitHub Actions.

## Initial Server Setup

1. Create a new Digital Ocean Droplet
   - Ubuntu 22.04 (LTS) recommended
   - Basic plan with 1 GB RAM is sufficient
   - Enable SSH key authentication during creation

2. Connect to your droplet via SSH:
   ```bash
   ssh root@your_droplet_ip
   ```

3. Create a non-root user with sudo privileges (for better security):
   ```bash
   # Create a new user
   adduser waterbot
   
   # Add user to sudo group
   usermod -aG sudo waterbot
   
   # Create .ssh directory for the new user
   mkdir -p /home/waterbot/.ssh
   
   # Copy your SSH key to the new user
   cp ~/.ssh/authorized_keys /home/waterbot/.ssh/
   
   # Set correct permissions
   chown -R waterbot:waterbot /home/waterbot/.ssh
   chmod 700 /home/waterbot/.ssh
   chmod 600 /home/waterbot/.ssh/authorized_keys
   ```

4. Update packages and install Docker & Docker Compose:
   ```bash
   apt update
   apt upgrade -y
   apt install -y docker.io docker-compose
   systemctl start docker
   systemctl enable docker
   ```

5. Add user to the Docker group to run Docker commands without sudo:
   ```bash
   usermod -aG docker waterbot
   ```

6. Create the application directory with correct permissions:
   ```bash
   mkdir -p /opt/water-bot
   chown -R waterbot:waterbot /opt/water-bot
   ```

7. Disconnect and reconnect using the new non-root user:
   ```bash
   exit
   ssh waterbot@your_droplet_ip
   ```

8. Verify Docker access:
   ```bash
   docker ps
   ```

## Creating a Deployment SSH Key

> For additional security recommendations, see [Security Recommendations](./security-recommendations.md)

It's best to create a dedicated SSH key for automated deployments rather than using your personal SSH key:

1. On your local machine, generate a dedicated deployment key:
   ```bash
   # Generate a new SSH key with no passphrase
   ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/water_bot_deploy
   
   # The output will be two files:
   # ~/.ssh/water_bot_deploy (private key)
   # ~/.ssh/water_bot_deploy.pub (public key)
   ```

2. Copy the public key to your server:
   ```bash
   # First, SSH to your server
   ssh waterbot@your_droplet_ip
   
   # Then, add to authorized_keys
   mkdir -p ~/.ssh
   nano ~/.ssh/authorized_keys
   
   # Paste the content of your local ~/.ssh/water_bot_deploy.pub file
   # Save and exit (Ctrl+X, then Y, then Enter)
   
   # Set proper permissions
   chmod 700 ~/.ssh
   chmod 600 ~/.ssh/authorized_keys
   ```

3. Test the new key:
   ```bash
   # From your local machine
   ssh -i ~/.ssh/water_bot_deploy waterbot@your_droplet_ip
   ```

## GitHub Repository Setup

1. Fork or clone the Water Bot repository to your GitHub account.

2. Add the following secrets to your GitHub repository:
   - Go to your GitHub repository → Settings → Secrets and variables → Actions
   - Add the following secrets:
     - `DIGITALOCEAN_HOST`: Your droplet's IP address (e.g., "123.45.67.89")
     - `DIGITALOCEAN_USERNAME`: Your non-root username (e.g., "waterbot")
     - `DIGITALOCEAN_PRIVATE_KEY`: The contents of your dedicated deployment private key, copied directly from the file:
       ```bash
       # Use this command to display your private key for copying
       cat ~/.ssh/water_bot_deploy
       ```
       Copy the ENTIRE output including the "-----BEGIN OPENSSH PRIVATE KEY-----" and "-----END OPENSSH PRIVATE KEY-----" lines.
     - `TELEGRAM_BOT_TOKEN`: Your Telegram bot token from BotFather
   
   > **Important**: All of these secrets must be set correctly for the deployment to work. The GitHub Actions workflow will validate that these secrets exist before attempting deployment.
   
   > **SSH Key Troubleshooting**: If you're having issues with the SSH key, ensure that:
   > 1. You've copied the ENTIRE private key file with all lines, not just a portion
   > 2. The key doesn't have a passphrase (generated with `-N ""`)
   > 3. The public key has been properly added to the server's authorized_keys file

## How the Deployment Works

The GitHub Actions workflow performs the following steps:

1. Checkout the repository code
2. Set up Docker Buildx
3. Build the Docker image
4. Copy the Docker image and docker-compose.yml to the server
5. Load the Docker image on the server
6. Create .env file with environment variables
7. Start the container with docker-compose

## Manual Deployment (If Needed)

If you need to deploy manually:

1. SSH into your Digital Ocean droplet:
   ```bash
   ssh waterbot@your_droplet_ip
   ```

2. Navigate to the application directory:
   ```bash
   cd /opt/water-bot
   ```

3. Pull the latest Docker image (if using a registry) or build locally:
   ```bash
   docker build -t water-bot:latest .
   ```

4. Create or update your .env file:
   ```bash
   echo "TELEGRAM_BOT_TOKEN=your_telegram_token" > .env
   ```

5. Start or restart the container:
   ```bash
   docker-compose down
   docker-compose up -d
   ```

## Troubleshooting

### Checking Container Logs

```bash
docker logs water-bot
```

### Manually Restarting the Container

```bash
docker-compose down
docker-compose up -d
```

### Inspecting Running Containers

```bash
docker ps
```

### Accessing the Container Shell

```bash
docker exec -it water-bot sh
```

### Sudo Access Issues

If you see permission issues:

```bash
# Make sure your non-root user has proper permissions
sudo chown -R waterbot:waterbot /opt/water-bot

# Verify Docker group membership
groups
# If docker is not listed, run:
sudo usermod -aG docker $(whoami)
# Then log out and back in
```

### SSH Key Issues

If GitHub Actions can't connect to your server:

1. Verify the format of your private key in GitHub Secrets:
   - Make sure you've included the entire key file content
   - Include the BEGIN and END lines

2. Check permissions on the server:
   ```bash
   # SSH to your server
   ssh waterbot@your_droplet_ip
   
   # Check permissions (should be 700 for .ssh and 600 for files)
   ls -la ~/.ssh
   
   # Fix if needed
   chmod 700 ~/.ssh
   chmod 600 ~/.ssh/authorized_keys
   ```

3. Test the SSH key locally:
   ```bash
   # From your development machine
   ssh -i ~/.ssh/water_bot_deploy -v waterbot@your_droplet_ip
   ```
   
4. Check SSH server logs for clues:
   ```bash
   # On your server
   sudo tail -f /var/log/auth.log
   ```