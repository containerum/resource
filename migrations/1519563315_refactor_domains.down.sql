CREATE TABLE IF NOT EXISTS new_domains (
  ip INET PRIMARY KEY,
  domain TEXT NOT NULL,
  domain_group TEXT NOT NULL DEFAULT 'system'
);

INSERT INTO new_domains
(ip, domain, domain_group)
SELECT unnest(d.ip), d.domain, d.domain_group FROM domains d;

DROP TABLE domains;

ALTER TABLE new_domains RENAME TO domains;