#!/bin/bash
# This script runs automatically when the Postgres container starts for the first time.
# It creates separate databases for order and inventory services.
# The auth_db is already created by the POSTGRES_DB environment variable.

set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE DATABASE order_db;
    CREATE DATABASE inventory_db;
    GRANT ALL PRIVILEGES ON DATABASE order_db TO retail;
    GRANT ALL PRIVILEGES ON DATABASE inventory_db TO retail;
EOSQL

echo "Databases created: auth_db, order_db, inventory_db"