DROP TRIGGER IF EXISTS check_container_resource_quotas ON containers;
DROP FUNCTION IF EXISTS check_container_resource_quotas();

DROP TRIGGER IF EXISTS check_deploy_resource_quotas ON deployments;
DROP FUNCTION IF EXISTS check_deploy_resource_quotas();