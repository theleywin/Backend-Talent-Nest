# 🎓 Explicación Visual: Network Alias y DNS en Docker

## 📊 Escenario 1: SIN Network Overlay (Problema)

```
Host Físico (IP: 192.168.1.100)
┌────────────────────────────────────────────┐
│                                            │
│  Frontend-1                                │
│  ├─ Puerto Host: 5173                     │
│  └─ Puerto Container: 5173                │
│                                            │
│  Frontend-2 ❌ FALLA                       │
│  ├─ Puerto Host: 5173 (ya ocupado!)       │
│  └─ Puerto Container: 5173                │
│                                            │
└────────────────────────────────────────────┘

ERROR: Bind for 0.0.0.0:5173 failed: port is already allocated
```

---

## ✅ Escenario 2: CON Network Overlay (Solución)

```
Host Físico (IP: 192.168.1.100)
┌────────────────────────────────────────────────────────────┐
│                                                            │
│  Docker Overlay Network "talentnet"                       │
│  ┌──────────────────────────────────────────────────┐    │
│  │                                                    │    │
│  │  Frontend-1                                       │    │
│  │  ├─ IP Overlay: 10.0.2.10 ← IP ÚNICA            │    │
│  │  ├─ Alias: "frontend"                            │    │
│  │  └─ Puerto: 5173 (en su propia red)             │    │
│  │                                                    │    │
│  │  Frontend-2                                       │    │
│  │  ├─ IP Overlay: 10.0.2.11 ← IP DIFERENTE        │    │
│  │  ├─ Alias: "frontend"                            │    │
│  │  └─ Puerto: 5173 (en su propia red)             │    │
│  │                                                    │    │
│  │  Frontend-3                                       │    │
│  │  ├─ IP Overlay: 10.0.2.12 ← IP DIFERENTE        │    │
│  │  ├─ Alias: "frontend"                            │    │
│  │  └─ Puerto: 5173 (en su propia red)             │    │
│  │                                                    │    │
│  └──────────────────────────────────────────────────┘    │
│                                                            │
└────────────────────────────────────────────────────────────┘

✅ NO HAY CONFLICTO: Cada contenedor usa su IP única
```

---

## 🔍 Cómo el Router Descubre los Frontends

### Paso 1: DNS Lookup

```go
// Router hace DNS lookup del alias "frontend"
ips, err := net.LookupIP("frontend")

// Docker DNS devuelve TODAS las IPs:
// ips = [10.0.2.10, 10.0.2.11, 10.0.2.12]
```

### Paso 2: Construcción de URLs

```go
for _, ip := range ips {
    ipStr := ip.String()
    
    // Construye URL única para cada IP
    targetURL := fmt.Sprintf("http://%s:5173", ipStr)
    
    // Resultado:
    // "http://10.0.2.10:5173"  → Frontend-1
    // "http://10.0.2.11:5173"  → Frontend-2  
    // "http://10.0.2.12:5173"  → Frontend-3
}
```

### Paso 3: Round Robin

```
Request 1 → Router → 10.0.2.10:5173 (Frontend-1)
Request 2 → Router → 10.0.2.11:5173 (Frontend-2)
Request 3 → Router → 10.0.2.12:5173 (Frontend-3)
Request 4 → Router → 10.0.2.10:5173 (Frontend-1) ← Reinicia ciclo
```

---

## 🎯 Comparación: Puerto Host vs Puerto Overlay

### ❌ INCORRECTO: Publicar puertos al host

```bash
docker run -d --network talentnet --network-alias frontend \
    -p 5173:5173 \  ← ❌ Conflicto
    frontend-1

docker run -d --network talentnet --network-alias frontend \
    -p 5173:5173 \  ← ❌ FALLA: puerto ya ocupado
    frontend-2
```

### ✅ CORRECTO: Solo red overlay (sin -p)

```bash
docker run -d --network talentnet --network-alias frontend \
    frontend-1  ← ✅ Escucha en 5173 INTERNAMENTE

docker run -d --network talentnet --network-alias frontend \
    frontend-2  ← ✅ Escucha en 5173 INTERNAMENTE

docker run -d --network talentnet --network-alias frontend \
    frontend-3  ← ✅ Escucha en 5173 INTERNAMENTE
```

**Resultado:**
```
Frontend-1: http://10.0.2.10:5173 ✅
Frontend-2: http://10.0.2.11:5173 ✅
Frontend-3: http://10.0.2.12:5173 ✅
```

---

## 🌐 Flujo Completo de una Request

```
1. Usuario → http://talentnest.com
                  ↓
2. DNS → IP del Router (192.168.1.100:80)
                  ↓
3. Router recibe request
                  ↓
4. Router hace DNS lookup: "frontend"
                  ↓
5. Docker DNS responde: [10.0.2.10, 10.0.2.11, 10.0.2.12]
                  ↓
6. Router selecciona: 10.0.2.11 (Round Robin)
                  ↓
7. Router proxy request → http://10.0.2.11:5173
                  ↓
8. Frontend-2 responde
                  ↓
9. Router retorna respuesta al usuario
```

---

## 🔬 Verificación Práctica

### Comando 1: Ver IPs de contenedores

```bash
# Inspeccionar red
docker network inspect talentnet

# Salida (ejemplo):
"Containers": {
    "abc123": {
        "Name": "frontend-1",
        "IPv4Address": "10.0.2.10/24",  ← IP única
    },
    "def456": {
        "Name": "frontend-2",
        "IPv4Address": "10.0.2.11/24",  ← IP única
    },
    "ghi789": {
        "Name": "frontend-3",
        "IPv4Address": "10.0.2.12/24",  ← IP única
    }
}
```

### Comando 2: DNS Lookup desde otro contenedor

```bash
# Crear contenedor temporal en la misma red
docker run --rm -it --network talentnet alpine sh

# Dentro del contenedor:
/ # nslookup frontend
Server:         127.0.0.11
Address:        127.0.0.11:53

Name:   frontend
Address: 10.0.2.10      ← Frontend-1
Address: 10.0.2.11      ← Frontend-2
Address: 10.0.2.12      ← Frontend-3
```

### Comando 3: Test de conectividad

```bash
# Desde el contenedor temporal
/ # wget -O- http://frontend:5173
# Conecta a UNO de los frontends (Docker DNS hace round-robin)

/ # wget -O- http://10.0.2.10:5173  # Específicamente Frontend-1
/ # wget -O- http://10.0.2.11:5173  # Específicamente Frontend-2
/ # wget -O- http://10.0.2.12:5173  # Específicamente Frontend-3
```

---

## 💡 Respuestas a tus Preguntas

### ❓ "¿Cómo obtengo las direcciones de los diferentes frontends?"

```go
// Docker DNS devuelve TODAS las IPs cuando haces lookup del alias
ips, _ := net.LookupIP("frontend")

// ips contiene: [10.0.2.10, 10.0.2.11, 10.0.2.12]
// Cada IP es un contenedor diferente
```

### ❓ "¿Cómo identifico dos contenedores con mismo host y mismo puerto?"

**Respuesta:** NO tienen el mismo "host":
- **Host físico:** Es el mismo (e.g., 192.168.1.100)
- **IP en red overlay:** Son DIFERENTES (10.0.2.10 vs 10.0.2.11)

El router NO usa el puerto del host físico, usa:
```go
targetURL := fmt.Sprintf("http://%s:5173", ipOverlay)
//                               ↑              ↑
//                         IP OVERLAY    Puerto interno
```

Cada contenedor tiene:
- ✅ Su propia IP en la red overlay (única)
- ✅ Su propio namespace de red (aislado)
- ✅ Puede usar el mismo puerto internamente sin conflicto

---

## 🎓 Analogía del Mundo Real

Imagina un edificio de apartamentos:

```
Edificio (Host Físico)
├─ Apartamento 101 (IP: 10.0.2.10)
│  └─ Puerta con número "5173"
│
├─ Apartamento 102 (IP: 10.0.2.11)
│  └─ Puerta con número "5173"  ← Mismo número, pero es OTRA puerta
│
└─ Apartamento 103 (IP: 10.0.2.12)
   └─ Puerta con número "5173"  ← Mismo número, pero es OTRA puerta
```

- **Dirección del edificio:** IP del host (192.168.1.100)
- **Apartamento:** IP en red overlay (10.0.2.10, etc.)
- **Número de puerta:** Puerto (5173)

Puedes tener múltiples "puertas 5173" porque están en apartamentos diferentes (IPs diferentes).

---

## ✅ Conclusión

1. **Docker asigna IPs únicas** a cada contenedor en la red overlay
2. **Network alias** permite que múltiples contenedores compartan un nombre DNS
3. **DNS lookup del alias** devuelve TODAS las IPs de contenedores con ese alias
4. **No hay conflicto de puertos** porque cada contenedor tiene su propia IP
5. **Router accede por IP overlay + puerto**, no por puerto del host

¿Necesitas que ejecute el demo script para que veas los IPs reales?
