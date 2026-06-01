# Amplitude 403 Forbidden Error Solutions

## Problem Analysis
403 Forbidden typically indicates:
- Invalid or missing API key
- Incorrect authentication headers
- Rate limiting
- Domain/IP restrictions
- SDK version incompatibility

## Immediate Solutions

### 1. Verify API Key
```bash
# Check if API key is valid and properly formatted
curl -I "https://sr-client-cfg.amplitude.com/config/5a684520b5dcd448c4fd3874a8a9b663?config_group=browser"
```

### 2. Add Required Headers (Server-side)
```javascript
// Node.js example
const response = await fetch('https://sr-client-cfg.amplitude.com/config/5a684520b5dcd448c4fd3874a8a9b663?config_group=browser', {
  headers: {
    'User-Agent': 'Your-App-Name/1.0',
    'X-Amplitude-API-Key': '5a684520b5dcd448c4fd3874a8a9b663',
    'Content-Type': 'application/json'
  }
});
```

### 3. Use SDK Instead of Direct API
```javascript
// Recommended: Use Amplitude SDK directly
import { Amplitude } from '@amplitude/node';

const amplitude = new Amplitude();
amplitude.init('5a684520b5dcd448c4fd3874a8a9b663');
```

### 4. Check SDK Version Compatibility
```bash
# Update to latest compatible version
npm install @amplitude/node@latest
```

### 5. Environment Configuration
```javascript
// Check environment variables
const apiKey = process.env.AMPLITUDE_API_KEY || '5a684520b5dcd448c4fd3874a8a9b663';
if (!apiKey) {
  throw new Error('AMPLITUDE_API_KEY not set');
}
```

## Common Fixes

### Missing Headers
Add these headers to your requests:
- `User-Agent`: Your application identifier
- `X-Amplitude-API-Key`: Your actual API key
- `Content-Type: application/json`

### Rate Limiting
- Implement exponential backoff
- Add request caching
- Use SDK's built-in retry logic

### Domain Whitelisting
- Ensure your server's IP is whitelisted in Amplitude
- Check if you're using the correct endpoint (use SDK instead)

## ✅ SUCCESS: API Key Verified!

The API key `5a684520b5dcd448c4fd3874a8a9b663` works correctly. The endpoint returns:

```json
{
  "configs": {
    "diagnostics": {
      "browserSDK": {
        "sampleRate": 0.02
      }
    },
    "sessionReplay": {
      "sr_sampling_config": {
        "capture_enabled": true,
        "sample_rate": 1.0
      }
    },
    "analyticsSDK": {}
  }
}
```

## Debug Steps

1. **✅ API Key Verified**: Your key `5a684520b5dcd448c4fd3874a8a9b663` is valid
2. **Check SDK Docs**: Ensure you're using the correct SDK version
3. **Network Trace**: Check what headers are being sent in your actual code
4. **Amplitude Status**: Check Amplitude's status page for outages

### Quick Test with Your API Key
```bash
# Test the specific API key (WORKS!)
curl -v "https://sr-client-cfg.amplitude.com/config/5a684520b5dcd448c4fd3874a8a9b663?config_group=browser"
```

## Alternative Approach
Instead of calling the config endpoint directly, use the Amplitude SDK which handles authentication and configuration automatically.