# CORS Configuration for Notify Chat Service

## 🌐 **Allowed Origins Configuration**

The Notify Chat Service has been configured to allow API calls from the following origins:

### **Default Allowed Origins:**
- `http://localhost:3000` - Local development frontend
- `https://localhost:3000` - Local development frontend (HTTPS)
- `https://notify-chat.netlify.app` - Production Netlify deployment
- `http://127.0.0.1:3000` - Alternative localhost address

### **Dynamic Origin Support:**
- **Any localhost variation** - For development and testing
- **Custom origins via environment variable** - Configure via `ALLOWED_ORIGINS`

## 🔧 **Configuration Methods**

### **1. Environment Variable (Recommended for Production)**
Set the `ALLOWED_ORIGINS` environment variable with comma-separated origins:

```bash
# .env or environment
ALLOWED_ORIGINS=http://localhost:3000,https://localhost:3000,https://notify-chat.netlify.app,https://mydomain.com

# Docker
docker run -e ALLOWED_ORIGINS="http://localhost:3000,https://mydomain.com" notify-chat-service

# Kubernetes
env:
  - name: ALLOWED_ORIGINS
    value: "http://localhost:3000,https://mydomain.com"
```

### **2. Code-Level Configuration**
The configuration is applied in two places:

**CORS Middleware** (`configs/middleware/cors.go`):
- Handles HTTP API requests
- Sets proper CORS headers
- Supports credentials

**WebSocket Upgrader** (`configs/utils/ws/upgrader.go`):
- Handles WebSocket connection upgrades
- Validates origin before upgrade

## 🚀 **Production Deployment Settings**

### **Environment Variables:**
```bash
# Production .env file
ALLOWED_ORIGINS=http://localhost:3000,https://localhost:3000,https://notify-chat.netlify.app
GIN_MODE=release
```

### **Docker Compose:**
```yaml
services:
  notify-chat-service:
    environment:
      - ALLOWED_ORIGINS=http://localhost:3000,https://notify-chat.netlify.app
      - GIN_MODE=release
```

### **Kubernetes:**
```yaml
env:
  - name: ALLOWED_ORIGINS
    value: "http://localhost:3000,https://notify-chat.netlify.app"
  - name: GIN_MODE
    value: "release"
```

## 📋 **Features**

### **CORS Headers Set:**
- `Access-Control-Allow-Origin`: Dynamic based on request origin
- `Access-Control-Allow-Methods`: GET, POST, PUT, DELETE, OPTIONS
- `Access-Control-Allow-Headers`: Origin, Content-Type, Authorization, X-Requested-With
- `Access-Control-Allow-Credentials`: true
- `Access-Control-Max-Age`: 86400 (24 hours)

### **Security Features:**
- ✅ **Origin Validation**: Only allows specified origins
- ✅ **Localhost Development**: Automatically allows localhost variations for development
- ✅ **WebSocket Security**: Same origin validation for WebSocket connections
- ✅ **Preflight Support**: Handles OPTIONS preflight requests
- ✅ **Credential Support**: Allows cookies and authorization headers

## 🧪 **Testing CORS**

### **Test Frontend Connection:**
From your frontend at `http://localhost:3000`:

```javascript
// Test API call
fetch('https://your-api-domain.com/api/health', {
  method: 'GET',
  credentials: 'include', // Important for CORS with credentials
  headers: {
    'Content-Type': 'application/json',
  }
})
.then(response => response.json())
.then(data => console.log('API Response:', data))
.catch(error => console.error('CORS Error:', error));

// Test WebSocket connection
const ws = new WebSocket('wss://your-api-domain.com/api/ws');
ws.onopen = () => console.log('WebSocket Connected');
ws.onerror = (error) => console.error('WebSocket Error:', error);
```

### **CORS Troubleshooting:**
1. **Check browser console** for CORS errors
2. **Verify Origin header** in network tab
3. **Ensure credentials are included** if using authentication
4. **Check preflight requests** (OPTIONS method)

## 🔄 **Dynamic Origin Updates**

To add new origins without code changes:

1. **Update environment variable:**
   ```bash
   ALLOWED_ORIGINS=http://localhost:3000,https://notify-chat.netlify.app,https://new-domain.com
   ```

2. **Restart the service:**
   ```bash
   # Docker
   docker restart notify-chat-service
   
   # Kubernetes
   kubectl rollout restart deployment/notify-chat-service
   ```

## 🌍 **Multi-Environment Support**

### **Development:**
```bash
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001,http://127.0.0.1:3000
```

### **Staging:**
```bash
ALLOWED_ORIGINS=http://localhost:3000,https://staging-notify-chat.netlify.app
```

### **Production:**
```bash
ALLOWED_ORIGINS=https://notify-chat.netlify.app,https://your-custom-domain.com
```

## ⚠️ **Security Notes**

- **Never use `*` for `Access-Control-Allow-Origin`** in production when credentials are enabled
- **Always validate origins** against a whitelist
- **Use HTTPS** in production for security
- **Regularly review** allowed origins list
- **Monitor** for unauthorized cross-origin requests

This configuration ensures your frontend applications can securely communicate with the Notify Chat Service API while maintaining proper security controls.
