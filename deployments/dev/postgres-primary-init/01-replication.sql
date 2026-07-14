-- Create the replicator role used for physical WAL streaming.
-- This is standard Postgres; no GitLab-specific SQL here.
DO $$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'replicator') THEN
      CREATE ROLE replicator WITH REPLICATION LOGIN PASSWORD 'replpass';
   END IF;
END
$$;

-- Network access for the replication user is controlled by pg_hba.conf
-- (see ./pg_hba.conf mounted into the init dir, or the postgres -c flags).
-- The dev compose network uses 172.16.0.0/12; production deployments
-- should restrict this to the actual secondary subnet.
GRANT CONNECT ON DATABASE gitlabhq_production TO replicator;