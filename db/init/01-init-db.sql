-- Initialize the Associate database
-- This script runs on first database initialization via docker-entrypoint-initdb.d

-- The AGE extension is already created by the apache/age image's built-in init script
-- (00-create-extension-age.sql), so we just need to ensure our graph exists

-- Create the associate graph if it doesn't exist
-- Note: This is also done by the application, but doing it here ensures
-- the database is ready before the app connects
DO $$
BEGIN
    -- Check if the graph already exists
    IF NOT EXISTS (
        SELECT 1 FROM ag_catalog.ag_graph WHERE name = 'associate'
    ) THEN
        PERFORM ag_catalog.create_graph('associate');
        RAISE NOTICE 'Created associate graph';
    ELSE
        RAISE NOTICE 'associate graph already exists';
    END IF;
END $$;
