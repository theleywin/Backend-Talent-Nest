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

### 8. Limpiar todo

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

## Próximos Pasos

1. ✅ Descubrimiento de nodos y elección de líder (COMPLETADO)
2. ⏳ Sistema de replicación líder-seguidor
3. ⏳ Sincronización de bases de datos SQLite
4. ⏳ Copia de datos a nuevos nodos
5. ⏳ Tolerancia a particiones de red
