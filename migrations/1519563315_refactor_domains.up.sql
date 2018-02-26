CREATE TABLE IF NOT EXISTS new_domains (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  domain TEXT NOT NULL UNIQUE,
  domain_group TEXT NOT NULL DEFAULT 'system',
  ip INET ARRAY NOT NULL
);

INSERT INTO new_domains
(domain, domain_group, ip)
SELECT d.domain, d.domain_group, array_agg(d.ip)
FROM domains d
GROUP BY d.domain, d.domain_group;

DROP TABLE domains;

ALTER TABLE new_domains RENAME TO domains;