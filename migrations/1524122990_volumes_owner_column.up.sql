ALTER TABLE volumes
  ADD COLUMN owner_user_id UUID;
UPDATE volumes SET owner_user_id = (
  SELECT permissions.owner_user_id
  FROM permissions
  WHERE (permissions.kind, permissions.resource_id) = ('volume',volumes.id) AND permissions.owner_user_id = permissions.user_id
)
WHERE NOT volumes.deleted;
UPDATE volumes SET owner_user_id = '00000000-0000-0000-0000-000000000000' WHERE owner_user_id IS NULL;
ALTER TABLE namespaces
  ALTER COLUMN owner_user_id SET NOT NULL;