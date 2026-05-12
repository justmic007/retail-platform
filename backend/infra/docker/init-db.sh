#!/bin/bash
# This script runs automatically when the Postgres container starts for the first time.
# It creates separate databases for order and inventory services.
# The auth_db is already created by the POSTGRES_DB environment variable.

set -e

# Connect to the default 'postgres' database (always exists) to create the others
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "postgres" <<-EOSQL
    CREATE DATABASE order_db;
    CREATE DATABASE inventory_db;
    GRANT ALL PRIVILEGES ON DATABASE order_db TO $POSTGRES_USER;
    GRANT ALL PRIVILEGES ON DATABASE inventory_db TO $POSTGRES_USER;
EOSQL

echo "Databases created: auth_db, order_db, inventory_db"