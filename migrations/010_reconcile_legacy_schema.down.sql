-- Intentionally irreversible.
--
-- This migration only restores schema objects expected by the application.
-- Dropping those columns or tables on rollback could destroy data that
-- predates the migration tracking table, so the safe down migration is a
-- no-op.
SELECT 1;
