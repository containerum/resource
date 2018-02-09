CREATE TABLE IF NOT EXISTS domains (
  ip INET PRIMARY KEY,
  domain TEXT NOT NULL,
  domain_group TEXT NOT NULL DEFAULT 'system'
);