# retail-platform

A production-grade microservices platform with a Next.js frontend.

## Structure

```
retail-platform/
├── backend/     # Go microservices — auth, order, inventory, notification
└── client/      # Next.js frontend
```

## Quick Start

**Backend:**
```bash
cd backend
make infra-up
make migrate-all
make run-auth
make run-inventory
make run-order
make run-notification
```

**Frontend:**
```bash
cd client
npm install
npm run dev
```

## Documentation

- [Backend README](./backend/README.md)
- [Auth Service](./backend/services/auth/README.md)
- [Order Service](./backend/services/order/README.md)
- [Inventory Service](./backend/services/inventory/README.md)
- [Notification Service](./backend/services/notification/README.md)
