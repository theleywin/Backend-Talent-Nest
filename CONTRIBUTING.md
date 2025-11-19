# TalentNest - Guía de Instalación con Docker

Esta guía te ayudará a desplegar toda la aplicación TalentNest usando Docker Swarm con MongoDB, Backend en Go y Frontend en React.

## 📋 Prerrequisitos

- Docker Desktop instalado y corriendo
- Git para clonar el repositorio

## 🚀 Pasos de Instalación

### 1. **Clonar el Repositorio**

```bash
git clone https://github.com/theleywin/Backend-Talent-Nest.git
cd Backend-Talent-Nest
```

### 2. **Inicializar Docker Swarm**

```bash
docker swarm init
```

### 3. **Crear Red Overlay**

```bash
docker network create --driver overlay --attachable talentnet
```

### 4. **Construir Imágenes Docker**

Antes de crear los servicios, necesitamos construir las imágenes del frontend y backend:

#### **a) Construir imagen del Backend (Go)**

```bash
# Navegar al directorio del backend
cd backend

# Construir imagen del backend
docker build -t backend-tn:latest .

# Verificar que la imagen se creó correctamente
docker images | grep backend-tn

# Volver al directorio raíz
cd ..
```

#### **b) Construir imagen del Frontend (React)**

```bash
# Navegar al directorio del frontend
cd frontend

# Construir imagen del frontend
docker build -t frontend-tn:latest .

# Verificar que la imagen se creó correctamente
docker images | grep frontend-tn

# Volver al directorio raíz
cd ..
```

#### **c) Construir imagen del Frontend (React)**

```bash
# Navegar al directorio del frontend
cd router

# Construir imagen del frontend
docker build -t router-tn:latest .

# Verificar que la imagen se creó correctamente
docker images | grep router-tn

# Volver al directorio raíz
cd ..
```

#### **d) Verificar todas las imágenes**

```bash
# Ver todas las imágenes creadas
docker images

# Deberías ver algo así:
# REPOSITORY      TAG       IMAGE ID       CREATED         SIZE
# backend-tn      latest    abc123def456   1 minute ago    XXX MB
# frontend-tn     latest    def456ghi789   2 minutes ago   XXX MB
# mongo           latest    ghi789jkl012   X days ago      XXX MB
```

### 5. **Desplegar MongoDB**

```bash
docker run --rm \
  --name mongodb \
  --network talentnet \
  -p 27017:27017 \
  mongo:latest
```

### 6. **Desplegar Backend (Go)**

```bash
# Crear servicio del backend usando la imagen construida previamente
docker run --rm \
  --name backend \
  --network talentnet \
  -p 3000:3000 \
  --env MONGO_URI=mongodb://mongodb:27017 \
  --env DB_NAME=databaseName \
  --env JWT_SECRET=secret_key \
  --env PORT=3000 \
  backend-tn:latest
```

### 7. **Desplegar Frontend (React)**

```bash
# Crear servicio del frontend usando la imagen construida previamente
docker run --rm \
  --name frontend \
  --network talentnet \
  -p 5173:5173 \
  frontend-tn:latest
```

### 7. **Desplegar Router (Go)**

```bash
# Crear servicio del router usando la imagen construida previamente
docker run -rm \
  --name router \
  --network talentnet \
  -p 8080:8080 \
  -e SERVICE_NAME=frontend \
  -e SERVICE_PORT=5173 \
  -e HEALTH_PATH=/ \
  router-tn:latest
```

## 🔍 Verificación

### Verificar que todos los servicios están corriendo:

```bash
docker service ls
```

Deberías ver algo así:
```
ID             NAME       MODE         REPLICAS   IMAGE              PORTS
xxxxx          mongodb    replicated   1/1        mongo:latest       *:27017->27017/tcp
xxxxx          backend    replicated   1/1        backend-tn:latest  *:3000->3000/tcp
xxxxx          frontend   replicated   1/1        frontend-tn:latest *:5173->5173/tcp
```

### Verificar el estado de cada servicio:

```bash
docker service ps mongodb
docker service ps backend
docker service ps frontend
```

### Ver logs de los servicios:

```bash
docker service logs mongodb
docker service logs backend
docker service logs frontend
```

## 🌐 Acceso a la Aplicación

Una vez que todos los servicios estén corriendo:

- **Frontend (React)**: http://localhost:5173
- **Backend API (Go)**: http://localhost:3000
- **MongoDB**: http://localhost:27017

## 🛠️ Comandos Útiles

### Actualizar un servicio:

```bash
# Reconstruir imagen
docker build -t backend-tn:latest .

# Actualizar servicio
docker service update --image backend-tn:latest backend
```

### Escalar servicios:

```bash
docker service scale backend=3
docker service scale frontend=2
```

### Eliminar servicios:

```bash
docker service rm frontend
docker service rm backend
docker service rm mongodb
```

### Eliminar red:

```bash
docker network rm talentnet
```


## 📝 Variables de Entorno

### Backend:
- `MONGO_URI`: URI de conexión a MongoDB
- `DB_NAME`: Nombre de la base de datos
- `JWT_SECRET`: Clave secreta para JWT
- `PORT`: Puerto del servidor


¡Tu aplicación TalentNest ahora está corriendo completamente en Docker Swarm! 🚀