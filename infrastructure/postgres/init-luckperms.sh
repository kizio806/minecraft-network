#!/bin/sh
set -eu

luckperms_pass="$(cat /run/secrets/luckperms_password)"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'luckperms_user') THEN
            CREATE ROLE luckperms_user WITH LOGIN PASSWORD '$luckperms_pass';
        ELSE
            ALTER ROLE luckperms_user WITH PASSWORD '$luckperms_pass';
        END IF;
    END
    \$\$;

    SELECT 'CREATE DATABASE luckperms OWNER luckperms_user'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'luckperms')\gexec

    REVOKE ALL ON DATABASE luckperms FROM PUBLIC;
    GRANT ALL PRIVILEGES ON DATABASE luckperms TO luckperms_user;
EOSQL
