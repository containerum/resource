ALTER TABLE deployments
  ADD COLUMN image TEXT CHECK (image != '');