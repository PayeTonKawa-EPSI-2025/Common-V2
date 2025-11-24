# Paye Ton Kawa - Infrastructure Setup

## Architecture Overview

This setup includes:
- **Traefik** (v3.2) - Ingress/Reverse Proxy
- **KrakenD** (v2.7) - API Gateway
- **RabbitMQ** (4.1) - Message Broker
- **3 Microservices**: Customers, Orders, Products

## Network Flow

```
Client Request
    ↓
Traefik (Port 80/443)
    ↓
KrakenD API Gateway (Port 8000)
    ↓
Microservices (Customers:8081, Orders:8082, Products:8083)
```

## Access Points

### Traefik Dashboard
- URL: http://localhost:8080
- Shows all registered services and routes

### KrakenD API Gateway
- URL: http://api.localhost (via Traefik)
- Direct: http://localhost:8000

### Microservices (Direct Access)
- Customers: http://customers.localhost (via Traefik)
- Orders: http://orders.localhost (via Traefik)
- Products: http://products.localhost (via Traefik)

### RabbitMQ Management
- URL: http://localhost:15672
- Credentials: guest/guest

## API Endpoints (via KrakenD)

### Customers API
- `GET http://api.localhost/customers` - List all customers
- `GET http://api.localhost/customers/{id}` - Get customer by ID
- `POST http://api.localhost/customers` - Create customer
- `PUT http://api.localhost/customers/{id}` - Update customer
- `DELETE http://api.localhost/customers/{id}` - Delete customer

### Orders API
- `GET http://api.localhost/orders` - List all orders
- `GET http://api.localhost/orders/{id}` - Get order by ID
- `POST http://api.localhost/orders` - Create order
- `PUT http://api.localhost/orders/{id}` - Update order
- `DELETE http://api.localhost/orders/{id}` - Delete order

### Products API
- `GET http://api.localhost/products` - List all products
- `GET http://api.localhost/products/{id}` - Get product by ID
- `POST http://api.localhost/products` - Create product
- `PUT http://api.localhost/products/{id}` - Update product
- `DELETE http://api.localhost/products/{id}` - Delete product

### Health Check
- `GET http://api.localhost/health` - KrakenD health status

## Starting the Services

### 1. Create the shared network (if not already created)
```bash
docker network create TPRE814
```

### 2. Start Common services (Traefik, KrakenD, RabbitMQ)
```bash
cd Common
docker compose up -d
```

### 3. Start microservices
```bash
# Customers
cd ../Customers
docker compose up -d

# Orders
cd ../Orders
docker compose up -d

# Products
cd ../Products
docker compose up -d
```

## Verifying the Setup

### Check all containers are running
```bash
docker ps
```

### Test via API Gateway
```bash
# Health check
curl http://api.localhost/health

# Get customers
curl http://api.localhost/customers

# Get orders
curl http://api.localhost/orders

# Get products
curl http://api.localhost/products
```

### Test direct access via Traefik
```bash
curl http://customers.localhost/customers
curl http://orders.localhost/orders
curl http://products.localhost/products
```

## Configuration Files

### Traefik
Configured via command-line arguments in `Common/compose.yaml`:
- Docker provider enabled
- HTTP endpoint on port 80
- HTTPS endpoint on port 443
- Dashboard on port 8080

### KrakenD
Configuration file: `Common/krakend/krakend.json`
- Aggregates all microservices
- Handles routing and load balancing
- Provides unified API interface
- Timeout: 3000ms
- Cache TTL: 300s

## Stopping the Services

```bash
# Stop all services
cd Common && docker compose down
cd ../Customers && docker compose down
cd ../Orders && docker compose down
cd ../Products && docker compose down
```

## Troubleshooting

### Check Traefik logs
```bash
docker logs traefik
```

### Check KrakenD logs
```bash
docker logs krakend
```

### Check if services are registered in Traefik
Visit http://localhost:8080 and check the HTTP routers section

### Verify network connectivity
```bash
docker network inspect TPRE814
```

## Notes

- All services use the shared `TPRE814` network for inter-service communication
- KrakenD references microservices by container name (customers-api, orders-api, products-api)
- Traefik auto-discovers services via Docker labels
- For production, configure HTTPS certificates and secure the Traefik dashboard
