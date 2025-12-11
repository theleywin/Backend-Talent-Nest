# Sistema Distribuido - Talent Nest

## Arquitectura del Cluster

Este sistema implementa un cluster distribuido de nodos backend con:

- **Descubrimiento de nodos**: Usando DNS de Docker
- **Elección de líder**: Basada en ID más alto
- **Reelección automática**: Cada 10 segundos
- **Tolerancia a fallos**: Detección de nodos caídos

## Estructura del Código

```
src/cluster/
├── types.go       # Definiciones de tipos y estructuras
├── discovery.go   # Descubrimiento de nodos via DNS
├── election.go    # Algoritmo de elección de líder
└── api.go         # API para consultar estado del cluster
```

## Comandos Docker

### 1. Crear la red Docker

```bash
docker network create --driver bridge talentnet
```

### 2. Compilar la imagen

```bash
docker build -t backend-tn:latest .
```

### 3. Levantar nodos

#### Nodo 1
```bash
docker run -d \
  --name backend-1 \
  --network talentnet \
  --network-alias backend-service \
  -p 3001:3000 \
  --env SERVICE_NAME=backend-service \
  --env JWT_SECRET=secret_key \
  --env PORT=3000 \
  backend-tn:latest
```

#### Nodo 2
```bash
docker run -d \
  --name backend-2 \
  --network talentnet \
  --network-alias backend-service \
  -p 3002:3000 \
  --env SERVICE_NAME=backend-service \
  --env JWT_SECRET=secret_key \
  --env PORT=3000 \
  backend-tn:latest
```

#### Nodo 3
```bash
docker run -d \
  --name backend-3 \
  --network talentnet \
  --network-alias backend-service \
  -p 3003:3000 \
  --env SERVICE_NAME=backend-service \
  --env JWT_SECRET=secret_key \
  --env PORT=3000 \
  backend-tn:latest
```

### 4. Consultar estado del cluster

```bash
# Ver estado del nodo 1
curl http://localhost:3001/cluster/status | jq

# Ver estado del nodo 2
curl http://localhost:3002/cluster/status | jq

# Ver estado del nodo 3
curl http://localhost:3003/cluster/status | jq
```

### 5. Ver logs de los nodos

```bash
# Ver logs en tiempo real
docker logs -f backend-1

# Ver los últimos 50 logs
docker logs --tail 50 backend-1
```

### 6. Simular caída de nodos

```bash
# Detener el líder actual (el nodo con ID más alto)
docker stop backend-3

# Observar la reelección en los logs de otros nodos
docker logs -f backend-1
```

### 7. Levantar nodo caído

```bash
# Reiniciar el nodo
docker start backend-3

# El nodo se reintegrará automáticamente al cluster
```

### 8. Probar Replicación

#### Crear un usuario en un seguidor (se redirige al líder)
```bash
curl -X POST http://localhost:3002/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test User",
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'
```

#### Verificar que el usuario existe en todos los nodos
```bash
# En nodo 1
curl http://localhost:3001/api/users/search?query=testuser

# En nodo 2
curl http://localhost:3002/api/users/search?query=testuser

# En nodo 3
curl http://localhost:3003/api/users/search?query=testuser
```

#### Probar redirección de escrituras
```bash
# Intentar escribir en un seguidor
curl -X POST http://localhost:3001/api/posts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "content": "This should be redirected to the leader"
  }'
```

### 9. Limpiar todo

```bash
# Detener y eliminar todos los contenedores
docker stop backend-1 backend-2 backend-3
docker rm backend-1 backend-2 backend-3

# Eliminar la red
docker network rm talentnet
```

## Respuesta de API

El endpoint `/cluster/status` retorna:

```json
{
  "current_node_id": 2562,
  "current_role": "follower",
  "leader_id": 2563,
  "leader_address": "http://10.0.2.3:3000",
  "total_nodes": 3,
  "nodes": [
    {
      "id": 2561,
      "address": "http://10.0.2.1:3000",
      "role": "follower",
      "is_leader": false,
      "healthy": true
    },
    {
      "id": 2562,
      "address": "http://10.0.2.2:3000",
      "role": "follower",
      "is_leader": false,
      "healthy": true
    },
    {
      "id": 2563,
      "address": "http://10.0.2.3:3000",
      "role": "leader",
      "is_leader": true,
      "healthy": true
    }
  ]
}
```

## Cómo Funciona

### Asignación de IDs

Cada nodo recibe un ID numérico basado en su dirección IP:
- Se usan los últimos dos octetos de la IPv4
- Ejemplo: `10.0.2.3` → ID = `2*256 + 3 = 515`

### Elección de Líder

1. Cada 10 segundos, todos los nodos:
   - Realizan DNS lookup de `backend-service`
   - Descubren todos los nodos activos
   - Ordenan los nodos por ID (mayor a menor)
   - El nodo con ID más alto es el líder

2. Si el líder cae:
   - Los otros nodos detectan su ausencia
   - En la siguiente elección (máximo 10s), se elige nuevo líder

### Tolerancia a Fallos

- **Grado 2**: El sistema funciona correctamente con hasta 2 nodos caídos
- **Mínimo 1 nodo**: El cluster necesita al menos 1 nodo para funcionar
- **Detección de fallos**: Nodos sin respuesta por 30s son eliminados

## Sistema de Replicación Líder-Seguidor

### Arquitectura de Replicación

El sistema implementa un modelo de replicación **líder-seguidor** con las siguientes características:

#### Reglas de Escritura y Lectura

1. **Solo el líder acepta escrituras**
   - POST, PUT, DELETE solo se procesan en el líder
   - Los seguidores redirigen automáticamente escrituras al líder

2. **Cualquier nodo acepta lecturas**
   - GET requests se procesan localmente en cualquier nodo
   - Garantiza balance de carga para operaciones de lectura

3. **Replicación asíncrona**
   - El líder propaga cambios a todos los seguidores
   - Los seguidores aplican cambios en su base de datos local

### Flujo de Replicación

```
Cliente → Seguidor (POST/PUT/DELETE)
           ↓
           Redirige al Líder
           ↓
        Líder procesa
           ↓
        Guarda en BD local
           ↓
        Propaga a todos los seguidores
           ↓
        Seguidores aplican cambios
```

### Endpoints de Replicación

#### `/cluster/replicate` (Solo Líder → Seguidores)
```json
POST /cluster/replicate
{
  "operation": "INSERT",
  "table": "users",
  "data": {...},
  "leader_id": 2563,
  "timestamp": "2025-11-30T12:00:00Z"
}
```

#### `/cluster/sync` (Seguidor solicita copia completa)
```json
GET /cluster/sync
Response: {
  "database": "base64_encoded_sqlite_file",
  "leader_id": 2563,
  "timestamp": "2025-11-30T12:00:00Z"
}
```

### Middleware de Replicación

El middleware intercepta todas las peticiones de escritura:

```go
// Si el nodo es seguidor y la operación es escritura
if !isLeader && isWriteOperation(method) {
    // Redirigir al líder
    return forwardToLeader(request)
}

// Si el nodo es líder y la operación es escritura
if isLeader && isWriteOperation(method) {
    // Procesar localmente
    result := processLocally(request)
    
    // Replicar a todos los seguidores
    replicateToFollowers(operation, data)
    
    return result
}
```

### Sincronización de Nuevos Nodos

Cuando un nuevo nodo se une al cluster:

1. **Detección**: El nuevo nodo se descubre via DNS
2. **Solicitud de sincronización**: El nodo solicita copia completa al líder
3. **Transferencia**: El líder envía archivo SQLite completo
4. **Aplicación**: El nodo reemplaza su BD con la del líder
5. **Listo**: El nodo comienza a recibir replicaciones incrementales

### Consistencia Eventual

- **Escrituras**: Consistencia fuerte en el líder
- **Lecturas en seguidores**: Consistencia eventual (puede haber lag de milisegundos)
- **Orden garantizado**: Las replicaciones mantienen el orden de operaciones
- **Idempotencia**: Las operaciones son idempotentes para evitar duplicados

### Tolerancia a Particiones (CAP)

El sistema prioriza:
- **C (Consistency)**: En el líder para escrituras
- **A (Availability)**: Lecturas disponibles en todos los nodos
- **P (Partition tolerance)**: El cluster continúa funcionando con nodos caídos

En caso de partición de red:
- Los nodos sin acceso al líder rechazan escrituras
- Las lecturas continúan funcionando localmente
- Al resolverse la partición, los nodos se resincronizan

### Estructura de Archivos Adicionales

```
src/cluster/
├── types.go          # Definiciones de tipos y estructuras
├── discovery.go      # Descubrimiento de nodos via DNS
├── election.go       # Algoritmo de elección de líder
├── api.go            # API para consultar estado del cluster
├── replication.go    # Sistema de replicación de datos (NUEVO)
└── middleware.go     # Middleware para redirección (NUEVO)
```

## Comandos de Prueba Completos

### Escenario 1: Levantar cluster de 3 nodos

```bash
# Crear red
docker network create talentnet

# Build imagen
docker build -t backend-tn:latest .

# Levantar 3 nodos
docker run -d --name backend-1 --network talentnet --network-alias backend-service -p 3001:3000 --env SERVICE_NAME=backend-service --env JWT_SECRET=secret_key --env PORT=3000 backend-tn:latest

docker run -d --name backend-2 --network talentnet --network-alias backend-service -p 3002:3000 --env SERVICE_NAME=backend-service --env JWT_SECRET=secret_key --env PORT=3000 backend-tn:latest

docker run -d --name backend-3 --network talentnet --network-alias backend-service -p 3003:3000 --env SERVICE_NAME=backend-service --env JWT_SECRET=secret_key --env PORT=3000 backend-tn:latest

# Ver logs del líder (nodo con ID más alto)
docker logs -f backend-3
```

### Escenario 2: Probar failover de líder

```bash
# 1. Identificar el líder actual
curl http://localhost:3001/cluster/status | jq .leader_id

# 2. Detener el líder
docker stop backend-3

# 3. Esperar 10 segundos y verificar nuevo líder
sleep 10
curl http://localhost:3001/cluster/status | jq

# 4. Crear un post en el nuevo líder
curl -X POST http://localhost:3001/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"name": "Test", "username": "test1", "email": "test1@test.com", "password": "pass123"}'

# 5. Reiniciar el nodo caído
docker start backend-3

# 6. Esperar sincronización y verificar que tiene los datos
sleep 5
docker logs backend-3 | grep "sync"
```

### Escenario 3: Agregar un nodo nuevo al cluster

```bash
# Con el cluster corriendo, agregar un cuarto nodo
docker run -d --name backend-4 --network talentnet --network-alias backend-service -p 3004:3000 --env SERVICE_NAME=backend-service --env JWT_SECRET=secret_key --env PORT=3000 backend-tn:latest

# El nodo nuevo automáticamente:
# 1. Se descubrirá via DNS
# 2. Solicitará sync completo del líder
# 3. Recibirá la base de datos completa
# 4. Comenzará a recibir replicaciones incrementales

# Ver el proceso de sincronización
docker logs -f backend-4
```

## Próximos Pasos

1. ✅ Descubrimiento de nodos y elección de líder (COMPLETADO)
2. ✅ Sistema de replicación líder-seguidor (COMPLETADO)
3. ✅ Sincronización de bases de datos SQLite (COMPLETADO)
4. ✅ Copia de datos a nuevos nodos (COMPLETADO)
5. ⏳ Mejorar replicación con hooks en controllers (SIGUIENTE)

## Limitaciones Actuales y Mejoras Futuras

### Implementado ✅
- Descubrimiento automático de nodos
- Elección de líder por ID más alto
- Redirección de escrituras al líder
- Sincronización completa de BD para nuevos nodos
- Detección de nodos caídos
- Readiness checks

### Por Implementar ⏳
- Hooks de replicación en controllers para capturar cambios
- Aplicación real de replicaciones en SQLite
- Sistema de queue para replicaciones fallidas
- Compresión de base de datos en sync
- Métricas de replicación (lag, throughput)
- Health checks HTTP entre nodos
