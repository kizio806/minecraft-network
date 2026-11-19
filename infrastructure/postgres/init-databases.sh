#!/bin/sh
set -eu

luckperms_pass="$(cat /run/secrets/luckperms_password)"
veloauth_pass="$(cat /run/secrets/veloauth_password)"
ultimateteams_pass="$(cat /run/secrets/ultimateteams_password 2>/dev/null || echo '')"
libertybans_pass="$(cat /run/secrets/libertybans_password 2>/dev/null || echo '')"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- LuckPerms user & database
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

if [ -f /docker-entrypoint-initdb.d/seed-luckperms.sql ]; then
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "luckperms" -f /docker-entrypoint-initdb.d/seed-luckperms.sql
elif [ -f "$(dirname "$0")/seed-luckperms.sql" ]; then
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "luckperms" -f "$(dirname "$0")/seed-luckperms.sql"
fi

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- VeloAuth user & database
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'veloauth_user') THEN
            CREATE ROLE veloauth_user WITH LOGIN PASSWORD '$veloauth_pass';
        ELSE
            ALTER ROLE veloauth_user WITH PASSWORD '$veloauth_pass';
        END IF;
    END
    \$\$;

    SELECT 'CREATE DATABASE veloauth OWNER veloauth_user'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'veloauth')\gexec

    REVOKE ALL ON DATABASE veloauth FROM PUBLIC;
    GRANT ALL PRIVILEGES ON DATABASE veloauth TO veloauth_user;
EOSQL

if [ -n "$ultimateteams_pass" ]; then
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
        -- UltimateTeams user & database
        DO \$\$
        BEGIN
            IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'ultimateteams_user') THEN
                CREATE ROLE ultimateteams_user WITH LOGIN PASSWORD '$ultimateteams_pass';
            ELSE
                ALTER ROLE ultimateteams_user WITH PASSWORD '$ultimateteams_pass';
            END IF;
        END
        \$\$;

        SELECT 'CREATE DATABASE ultimateteams OWNER ultimateteams_user'
        WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'ultimateteams')\gexec

        REVOKE ALL ON DATABASE ultimateteams FROM PUBLIC;
        GRANT ALL PRIVILEGES ON DATABASE ultimateteams TO ultimateteams_user;
	EOSQL
fi

if [ -n "$libertybans_pass" ]; then
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
        -- LibertyBans user & database
        DO \$\$
        BEGIN
            IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'libertybans_user') THEN
                CREATE ROLE libertybans_user WITH LOGIN PASSWORD '$libertybans_pass';
            ELSE
                ALTER ROLE libertybans_user WITH PASSWORD '$libertybans_pass';
            END IF;
        END
        \$\$;

        SELECT 'CREATE DATABASE libertybans OWNER libertybans_user'
        WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'libertybans')\gexec

        REVOKE ALL ON DATABASE libertybans FROM PUBLIC;
        GRANT ALL PRIVILEGES ON DATABASE libertybans TO libertybans_user;
	EOSQL
fi
