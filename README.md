# Backend TalentNest

Backend API para TalentNest desarrollado en Go con MongoDB.

## Prerrequisitos

- Go 1.19 o superior
- MongoDB (local o Atlas)
- Git

## Instalación

### 1. Clonar el repositorio

```bash
git clone https://github.com/theleywin/Backend-Talent-Nest.git
cd Backend-Talent-Nest
```

### 2. Instalar dependencias

```bash
go mod tidy
```

### 3. Configurar variables de entorno

Crear archivo `.env` en la raíz del proyecto:

```env
MONGO_URI=mongodb://localhost:27017
DB_NAME=databaseName
JWT_SECRET=secret_key
PORT=3000
```

## Iniciar el servidor

### Modo desarrollo

```bash
go run .
```

### Compilar y ejecutar

```bash
go build -o bin/talentnest main.go
./bin/talentnest
```