-- remove all non owners access records if we remove owner`s access record
CREATE OR REPLACE FUNCTION remove_users_on_remove_owner() RETURNS TRIGGER AS $remove_users_on_remove_owner$
BEGIN
  IF OLD.user_id = OLD.owner_user_id THEN
    DELETE FROM permissions WHERE (kind, resource_id) = (OLD.kind, OLD.resource_id);
  END IF;
  RETURN NULL;
END;
$remove_users_on_remove_owner$ LANGUAGE plpgsql;