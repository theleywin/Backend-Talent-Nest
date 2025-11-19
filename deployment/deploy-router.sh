#!/bin/bash

#############################################
# Deploy Router - DNS Service Discovery
# Routes external traffic to healthy frontends
#############################################

set -e

# Configuration
NETWORK="talentnet"
IMAGE_NAME="router-tn"
IMAGE_TAG="latest"
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"
CONTAINER_NAME="router"
ROUTER_PORT=8080
SERVICE_NAME="frontend"
SERVICE_PORT="5173"
HEALTH_PATH="/"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Router Deployment - DNS Discovery${NC}"
echo -e "${BLUE}========================================${NC}"

# Check if network exists
echo -e "\n${YELLOW}📡 Checking overlay network...${NC}"
if ! docker network ls | grep -q "$NETWORK"; then
    echo -e "${YELLOW}Creating overlay network: $NETWORK${NC}"
    docker network create --driver overlay --attachable $NETWORK
    echo -e "${GREEN}✅ Network created${NC}"
else
    echo -e "${GREEN}✅ Network exists${NC}"
fi

# Build router image
echo -e "\n${YELLOW}📦 Building router image...${NC}"
cd ../router || exit 1

if [ ! -f "Dockerfile" ]; then
    echo -e "${RED}❌ Error: Dockerfile not found in router directory${NC}"
    exit 1
fi

docker build -t $FULL_IMAGE .
echo -e "${GREEN}✅ Image built: $FULL_IMAGE${NC}"

cd ../deployment || exit 1

# Stop and remove existing router
echo -e "\n${YELLOW}🧹 Cleaning up existing router...${NC}"
if docker ps -a | grep -q "$CONTAINER_NAME"; then
    echo -e "Removing $CONTAINER_NAME..."
    docker stop $CONTAINER_NAME 2>/dev/null || true
    docker rm $CONTAINER_NAME 2>/dev/null || true
fi
echo -e "${GREEN}✅ Cleanup complete${NC}"

# Deploy router
echo -e "\n${YELLOW}🚀 Deploying router...${NC}"

docker run -d \
    --name $CONTAINER_NAME \
    --network $NETWORK \
    -p ${ROUTER_PORT}:8080 \
    -e SERVICE_NAME=$SERVICE_NAME \
    -e SERVICE_PORT=$SERVICE_PORT \
    -e HEALTH_PATH=$HEALTH_PATH \
    -e ROUTER_PORT=8080 \
    --restart unless-stopped \
    --label "service=router" \
    $FULL_IMAGE

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Router deployed successfully${NC}"
else
    echo -e "${RED}❌ Failed to deploy router${NC}"
    exit 1
fi

# Wait for router to start
echo -e "\n${YELLOW}⏳ Waiting for router to start...${NC}"
sleep 5

# Verify deployment
echo -e "\n${YELLOW}🔍 Verifying deployment...${NC}"

RUNNING=$(docker ps --filter "name=$CONTAINER_NAME" --format "{{.Names}}" | wc -l | xargs)

if [ "$RUNNING" -eq 1 ]; then
    echo -e "${GREEN}✅ Router is running${NC}"
else
    echo -e "${RED}❌ Router is not running${NC}"
    exit 1
fi

# Show container info
echo -e "\n${BLUE}Running Router Container:${NC}"
docker ps --filter "name=$CONTAINER_NAME" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Health check
echo -e "\n${YELLOW}🏥 Performing health check...${NC}"
sleep 3

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:${ROUTER_PORT}/router/health 2>/dev/null || echo "000")

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✅ Router health check passed${NC}"
else
    echo -e "${YELLOW}⚠️  Router health check returned: HTTP $HTTP_CODE${NC}"
fi

# Get router status
echo -e "\n${YELLOW}📊 Fetching router status...${NC}"
curl -s http://localhost:${ROUTER_PORT}/router/status | python3 -m json.tool 2>/dev/null || \
curl -s http://localhost:${ROUTER_PORT}/router/status

# View logs
echo -e "\n${YELLOW}📋 Recent logs:${NC}"
docker logs --tail 20 $CONTAINER_NAME

# Final status
echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}  Deployment Complete${NC}"
echo -e "${BLUE}========================================${NC}"

echo -e "\n${GREEN}📊 Router Information:${NC}"
echo -e "  - Container: $CONTAINER_NAME"
echo -e "  - Image: $FULL_IMAGE"
echo -e "  - Port: $ROUTER_PORT"
echo -e "  - Target Service: $SERVICE_NAME"
echo -e "  - Network: $NETWORK"

echo -e "\n${BLUE}📍 Access Points:${NC}"
echo -e "  - Router Proxy: http://localhost:${ROUTER_PORT}"
echo -e "  - Router Status: http://localhost:${ROUTER_PORT}/router/status"
echo -e "  - Router Health: http://localhost:${ROUTER_PORT}/router/health"

echo -e "\n${BLUE}🔧 Management Commands:${NC}"
echo -e "  - View logs: docker logs -f $CONTAINER_NAME"
echo -e "  - Stop: docker stop $CONTAINER_NAME"
echo -e "  - Restart: docker restart $CONTAINER_NAME"
echo -e "  - Remove: docker rm -f $CONTAINER_NAME"

echo -e "\n${BLUE}🌐 DNS Configuration:${NC}"
echo -e "  - Add to /etc/hosts: 127.0.0.1 talentnest.com"
echo -e "  - Then access: http://talentnest.com:${ROUTER_PORT}"

echo -e "\n${GREEN}🎉 Router deployment complete!${NC}"
echo -e "${GREEN}The router will automatically discover and route to healthy frontends every 10 seconds${NC}"
