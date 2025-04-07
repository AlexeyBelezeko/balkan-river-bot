# Security Recommendations for Water Bot Deployment

This document outlines security best practices for deploying the Water Bot service.

## Deployment Security

### SSH Access

1. **Use Dedicated Deployment Keys**
   - Generate a dedicated SSH key pair for GitHub Actions deployments
   - Never use your personal SSH keys for automated deployments
   - Limit the scope of deployment keys to only necessary actions

2. **Disable Password Authentication**
   ```bash
   # Edit SSH config file
   sudo nano /etc/ssh/sshd_config
   
   # Set these values
   PasswordAuthentication no
   ChallengeResponseAuthentication no
   UsePAM no
   
   # Restart SSH service
   sudo systemctl restart sshd
   ```

3. **Consider SSH Key Restrictions**
   You can add restrictions to the SSH keys in authorized_keys:
   ```
   command="cd /opt/water-bot && docker-compose up -d",no-port-forwarding,no-X11-forwarding ssh-ed25519 AAAAC3NzaC1lZDI1... github-actions-deploy
   ```

### User Management

1. **Use Non-Root Users**
   - Always run the service and deploy as a non-root user
   - Limit sudo access to only necessary commands

2. **Create a Service-Specific User**
   - The waterbot user should have minimal privileges
   - Apply the principle of least privilege

### Network Security

1. **Use a Firewall**
   ```bash
   # Install UFW (Uncomplicated Firewall)
   sudo apt install ufw
   
   # Allow SSH
   sudo ufw allow ssh
   
   # Enable UFW
   sudo ufw enable
   ```

2. **Regularly Update System**
   ```bash
   sudo apt update
   sudo apt upgrade
   ```

## Docker Security

1. **Keep Images Updated**
   - Regularly update base images to include security patches
   - Use specific versions rather than 'latest' tag

2. **Minimize Container Capabilities**
   Update docker-compose.yml to include:
   ```yaml
   services:
     water-bot:
       # Other settings...
       security_opt:
         - no-new-privileges:true
       read_only: true
       tmpfs:
         - /tmp
   ```

3. **Scan Docker Images**
   ```bash
   # Install trivy
   sudo apt install trivy
   
   # Scan image
   trivy image water-bot:latest
   ```

## Environment Variables

1. **Protect Telegram Bot Token**
   - Never commit tokens to the repository
   - Always use environment variables or secrets

2. **Use .env File with Restricted Permissions**
   ```bash
   touch /opt/water-bot/.env
   chmod 600 /opt/water-bot/.env
   chown waterbot:waterbot /opt/water-bot/.env
   ```

## Monitoring and Logging

1. **Enable Logging**
   - Ensure the application logs important events
   - Consider forwarding logs to a centralized system

2. **Set Up Monitoring**
   - Monitor system resources
   - Set up alerts for unusual activity

## Regular Maintenance

1. **Backup Configuration**
   - Regularly backup the .env file and docker-compose.yml

2. **Test Recovery Procedures**
   - Periodically test that you can rebuild the deployment

3. **Rotate SSH Keys**
   - Regularly rotate the deployment SSH keys
   - Update the GitHub secrets when keys are rotated