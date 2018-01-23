DROP TRIGGER IF EXISTS remove_volume_perms ON volumes;
DROP FUNCTION IF EXISTS remove_volume_perms();

DROP TRIGGER IF EXISTS remove_namespace_perms ON namespaces;
DROP FUNCTION IF EXISTS remove_namespace_perms();

DROP TRIGGER IF EXISTS remove_users_on_remove_owner ON permissions;
DROP FUNCTION IF EXISTS remove_users_on_remove_owner();

DROP TRIGGER IF EXISTS update_users_permissions ON permissions;
DROP FUNCTION IF EXISTS update_users_permissions();

DROP TRIGGER IF EXISTS insert_owner_permissions ON permissions;
DROP FUNCTION IF EXISTS insert_owner_permissions();

DROP TRIGGER IF EXISTS check_update_permissions ON permissions;
DROP FUNCTION IF EXISTS check_update_permissions();

DROP TRIGGER IF EXISTS check_resource_id ON permissions;
DROP FUNCTION IF EXISTS check_resource_id();

DROP TABLE IF EXISTS permissions;

DROP TYPE IF EXISTS PERMISSION_STATUS;
DROP TYPE IF EXISTS RESOURCE_KIND;