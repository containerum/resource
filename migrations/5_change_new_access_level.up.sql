BEGIN TRANSACTION;
ALTER TABLE accesses ADD COLUMN new_access_level AccessLevel NULL;
ALTER TABLE accesses DROP CONSTRAINT accesses_check;
ALTER TABLE accesses DROP CONSTRAINT accesses_check1;
UPDATE accesses SET limited=false WHERE limited IS NULL;
ALTER TABLE accesses ALTER COLUMN limited SET NOT NULL;
COMMIT TRANSACTION;
