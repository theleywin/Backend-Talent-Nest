# 🌐 TalentNest Router - DNS Service Discovery

Router inteligente en Go para Docker Swarm que descubre automáticamente contenedores frontend activos usando DNS de Docker y network aliases.

## 🏗️ Arquitectura

```
Navegador (talentnest.com:8080)
         ↓
    Router (Go)
    - Descubrimiento DNS cada 10s
    - Health checks cada 5s
    - Round Robin Load Balancing
         ↓
    Docker Swarm DNS
    - Resuelve "frontend" → IPs
         ↓
    Frontends (con network alias "frontend")
    - frontend-1: 10.0.1.2:5173
    - frontend-2: 10.0.1.3:5173
    - frontend-3: 10.0.1.4:5173
```

## ⚙️ Funcionamiento

### 1. **Descubrimiento DNS (cada 10 segundos)**
El router usa `net.LookupIP("frontend")` para obtener todas las IPs de contenedores con el alias "frontend" en la red overlay.

```go
// En goroutine separada
ips, err := net.LookupIP("frontend")
// Retorna: [10.0.1.2, 10.0.1.3, 10.0.1.4]
```

### 2. **Health Checks (cada 5 segundos)**
Para cada IP descubierta, el router hace un GET request al health endpoint (por defecto `/`).

```go
// En goroutine paralela para cada endpoint
resp, err := client.Get("http://10.0.1.2:5173/")
if resp.StatusCode == 200 {
    endpoint.IsHealthy = true
}
```

### 3. **Load Balancing (Round Robin)**
Cuando llega una request, el router selecciona el siguiente endpoint saludable:

```go
healthyEndpoints := filterHealthy(endpoints)
selected := healthyEndpoints[currentIndex % len(healthyEndpoints)]
proxy.ServeHTTP(w, r) // Forward to selected
```

## 🚀 Despliegue

### Prerrequisitos
1. Docker Swarm inicializado
2. Red overlay `talentnet` creada
3. Frontends desplegados con network alias

### Paso 1: Construir Imagen

```bash
cd router
docker build -t router-tn:latest .
```

### Paso 2: Desplegar Router

```bash
# Opción 1: Usando el script
cd ../deployment
chmod +x deploy-router.sh
./deploy-router.sh

# Opción 2: Manual
docker run -d \
    --name router \
    --network talentnet \
    -p 8080:8080 \
    -e SERVICE_NAME=frontend \
    -e SERVICE_PORT=5173 \
    -e HEALTH_PATH=/ \
    router-tn:latest
```

### Paso 3: Configurar DNS Local (macOS/Linux)

```bash
# Agregar a /etc/hosts
sudo nano /etc/hosts

# Añadir línea:
127.0.0.1 talentnest.com
```

### Paso 4: Acceder

```bash
# Abrir navegador
http://talentnest.com:8080
```

## 📋 Variables de Entorno

| Variable | Descripción | Default |
|----------|-------------|---------|
| `SERVICE_NAME` | Network alias del servicio a enrutar | `frontend` |
| `SERVICE_PORT` | Puerto del servicio target | `5173` |
| `HEALTH_PATH` | Path para health check | `/` |
| `ROUTER_PORT` | Puerto donde escucha el router | `8080` |

## 🔍 Endpoints del Router

### Proxy
```bash
# Redirige a frontend saludable
GET http://localhost:8080/
GET http://localhost:8080/cualquier/ruta
```

### Status
```bash
# Información de endpoints descubiertos
GET http://localhost:8080/router/status

# Respuesta:
{
  "service": "frontend",
  "total": 3,
  "healthy": 2,
  "endpoints": [
    {
      "ip": "10.0.1.2",
      "url": "http://10.0.1.2:5173",
      "is_healthy": true,
      "last_check": "2024-11-18T10:30:00Z"
    },
    ...
  ]
}
```

### Health
```bash
# Health del router mismo
GET http://localhost:8080/router/health
# Response: OK (200)
```

## 📊 Monitoreo

### Ver Logs
```bash
docker logs -f router
```

**Logs esperados:**
```
🚀 Starting Router Manager for service: frontend
🔍 Discovering frontends via DNS: frontend
📡 Discovered 3 frontend IPs
✅ New frontend discovered: http://10.0.1.2:5173
✅ New frontend discovered: http://10.0.1.3:5173
✅ New frontend discovered: http://10.0.1.4:5173
🏥 Health Check: 3/3 frontends healthy
🔀 Proxying request to: http://10.0.1.2:5173
🔀 Proxying request to: http://10.0.1.3:5173
```

### Status en Tiempo Real
```bash
# Monitoreo continuo
watch -n 2 'curl -s http://localhost:8080/router/status | jq'
```

## 🧪 Testing

### Test 1: Descubrimiento Automático

```bash
# 1. Desplegar frontends con network alias
docker run -d --name frontend-1 --network talentnet --network-alias frontend -p 5173:5173 frontend-tn:latest
docker run -d --name frontend-2 --network talentnet --network-alias frontend -p 5174:5173 frontend-tn:latest
docker run -d --name frontend-3 --network talentnet --network-alias frontend -p 5175:5173 frontend-tn:latest

# 2. Desplegar router
./deploy-router.sh

# 3. Verificar descubrimiento
curl http://localhost:8080/router/status

# Deberías ver 3 endpoints
```

### Test 2: Health Checks

```bash
# 1. Detener un frontend
docker stop frontend-2

# 2. Esperar 5-10 segundos (ciclo de health check)
sleep 10

# 3. Verificar status
curl http://localhost:8080/router/status

# frontend-2 debería aparecer como unhealthy
# El router ya no enviará tráfico ahí
```

### Test 3: Load Balancing

```bash
# Hacer múltiples requests
for i in {1..10}; do
    curl -s http://localhost:8080/ | head -1
    echo "Request $i sent"
done

# Ver logs del router para verificar distribución
docker logs router | grep "Proxying request"

# Deberías ver distribución entre frontends healthy
```

### Test 4: Recuperación Automática

```bash
# 1. Reiniciar frontend detenido
docker start frontend-2

# 2. Esperar descubrimiento (10s + health check 5s)
sleep 15

# 3. Verificar que vuelve a recibir tráfico
curl http://localhost:8080/router/status

# frontend-2 debería aparecer como healthy de nuevo
```

## 🔧 Troubleshooting

### Problema: Router no descubre frontends

**Síntoma:**
```json
{
  "total": 0,
  "healthy": 0
}
```

**Solución:**
```bash
# 1. Verificar que frontends tengan network alias
docker inspect frontend-1 | grep -A 5 "Networks"

# Debe mostrar:
# "talentnet": {
#   "Aliases": ["frontend", ...]
# }

# 2. Verificar que router esté en la misma red
docker inspect router | grep -A 5 "Networks"

# 3. Test DNS manualmente desde el router
docker exec -it router nslookup frontend
```

### Problema: Frontends marcados como unhealthy

**Síntoma:**
```
🏥 Health Check: 0/3 frontends healthy
```

**Solución:**
```bash
# 1. Verificar que frontends responden al health path
curl http://localhost:5173/

# 2. Revisar health path configurado
docker inspect router | grep HEALTH_PATH

# 3. Cambiar health path si es necesario
docker rm -f router
docker run -d --name router --network talentnet -p 8080:8080 \
    -e HEALTH_PATH=/health \
    router-tn:latest
```

### Problema: DNS no resuelve

**Síntoma:**
```
❌ DNS lookup failed for frontend: no such host
```

**Solución:**
```bash
# 1. Verificar que la red overlay existe
docker network ls | grep talentnet

# 2. Crear si no existe
docker network create --driver overlay --attachable talentnet

# 3. Asegurar que frontends y router están en la red
docker network inspect talentnet
```

## 🎯 Configuración de Network Alias en Frontends

Para que el router pueda descubrir los frontends, estos deben tener el network alias configurado:

```bash
# Opción 1: Al crear el contenedor
docker run -d \
    --name frontend-1 \
    --network talentnet \
    --network-alias frontend \
    -p 5173:5173 \
    frontend-tn:latest

# Opción 2: Conectar a red existente con alias
docker network connect --alias frontend talentnet frontend-1

# Verificar alias
docker inspect frontend-1 | grep -A 10 "Networks"
```

## 🔒 Producción

Para uso en producción:

1. **HTTPS**: Agregar soporte TLS al router
2. **Métricas**: Implementar Prometheus metrics
3. **Logging**: Estructurar logs en JSON
4. **Configuración**: Usar archivos de config externos
5. **Seguridad**: Implementar rate limiting y authentication

## 📚 Referencias

- Docker Swarm DNS: https://docs.docker.com/engine/swarm/networking/
- Network Aliases: https://docs.docker.com/network/
- Go net package: https://pkg.go.dev/net

---

**Desarrollado para TalentNest - Sistema Distribuido con FT=2**
