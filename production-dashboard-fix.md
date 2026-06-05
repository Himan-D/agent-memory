# Production Dashboard 500 Error Troubleshooting

## Issue Summary
- **URL**: https://dashboard.hystersis.ai/
- **Error**: 500 Internal Server Error
- **Status**: Production deployment issue

## Common Causes & Solutions

### 1. Stale Build Artifacts (Most Likely)
```bash
# SSH into production server
ssh user@your-server

# Navigate to dashboard directory
cd /path/to/dashboard

# Clear build cache
rm -rf .next

# Rebuild
npm run build

# Restart service
pm2 restart dashboard  # or whichever process manager
```

### 2. Environment Variables Missing
Check these required environment variables in production:
```bash
NEXT_PUBLIC_API_URL=https://api.hystersis.ai
NEXTAUTH_URL=https://dashboard.hystersis.ai
NEXTAUTH_SECRET=your-production-secret
ADMIN_API_KEY=your-production-admin-key
NEXT_PUBLIC_AMPLITUDE_API_KEY=5a684520b5dcd448c4fd3874a8a9b663
```

### 3. Backend API Not Running
Ensure backend API is accessible:
```bash
curl -s https://api.hystersis.ai/health
```
Should return: `{"status":"ok"}`

### 4. Dependencies Installation
```bash
# Install dependencies
npm install

# Or if using CI/CD, ensure package-lock.json is included
```

### 5. Build Process Issues
```bash
# Clean install and build
rm -rf node_modules .next
npm install
npm run build
```

## Debug Steps

### Check Server Logs
```bash
# Check PM2 logs
pm2 logs dashboard

# Check system logs
journalctl -u your-service-name -f

# Check nginx logs (if using nginx)
tail -f /var/log/nginx/error.log
```

### Check Specific Error Details
```bash
# Enable debug mode temporarily
export DEBUG=*
pm2 reload dashboard

# Or check PM2 restart logs
pm2 restart dashboard --watch
```

### Verify Configuration
```bash
# Check environment variables in production
echo $NEXT_PUBLIC_API_URL
echo $NEXTAUTH_URL
echo $NEXTAUTH_SECRET
```

## Quick Fix Commands

### If using PM2
```bash
pm2 stop dashboard
pm2 delete dashboard
rm -rf .next
npm run build
pm2 start ecosystem.config.js --name dashboard
pm2 save
```

### If using Docker
```bash
docker-compose down
docker-compose up -d --build
```

### If using systemd
```bash
sudo systemctl stop dashboard
sudo rm -rf /var/www/dashboard/.next
npm run build
sudo systemctl start dashboard
sudo systemctl status dashboard
```

## Prevention Tips

1. **Always clean build cache** before deploying
2. **Verify environment variables** are set correctly
3. **Test backend API connectivity** before deploying frontend
4. **Monitor logs** during deployment
5. **Use health checks** in your deployment script

## Contact Support
If these steps don't resolve the issue, check:
- Production server logs
- Deployment pipeline logs
- Network connectivity to backend API
- SSL certificate status