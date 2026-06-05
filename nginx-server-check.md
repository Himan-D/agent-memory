# Nginx & Server Diagnostic Commands

## Check Nginx Status
```bash
# Check nginx service status
sudo systemctl status nginx

# Check if nginx is running
sudo nginx -t

# Check nginx configuration
sudo nginx -t

# View nginx error logs
sudo tail -f /var/log/nginx/error.log

# View nginx access logs
sudo tail -f /var/log/nginx/access.log

# Check nginx process
ps aux | grep nginx
```

## Check Dashboard Service
```bash
# Check PM2 status for dashboard
pm2 status

# Check PM2 logs for dashboard
pm2 logs dashboard

# Check if dashboard process is running
ps aux | grep node

# Check dashboard logs
pm2 logs dashboard --lines 50
```

## Check System Resources
```bash
# Check disk space
df -h

# Check memory usage
free -h

# Check CPU usage
top

# Check active processes
ps aux --sort=-%cpu | head -10
```

## Check Network Ports
```bash
# Check what's listening on port 80/443
sudo ss -tlnp | grep ':80\|:443'

# Check if port 80/443 is accessible
curl -I http://localhost:80
curl -I https://localhost:443
```

## Quick Fix Commands
```bash
# Restart nginx
sudo systemctl restart nginx

# Reload nginx configuration
sudo systemctl reload nginx

# Check if nginx can reach dashboard
curl http://localhost:3001
```

## Common Issues to Check

1. **Nginx Configuration**: Ensure proxy to localhost:3001 is correct
2. **Dashboard Service**: Verify dashboard is running on port 3001
3. **SSL Certificate**: Check if HTTPS is properly configured
4. **Firewall**: Ensure ports 80/443 are open
5. **Memory/Disk**: Check if server is out of resources

Run these commands and share the output so I can help diagnose the specific issue.