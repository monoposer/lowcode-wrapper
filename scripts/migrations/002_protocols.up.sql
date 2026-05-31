ALTER TABLE wrapper_server DROP CONSTRAINT IF EXISTS wrapper_server_protocol_check;

ALTER TABLE wrapper_server ADD CONSTRAINT wrapper_server_protocol_check
  CHECK (protocol IN (
    'http', 'postgres', 'mysql', 'file',
    'mongo', 's3', 'firebase', 'notion', 'redis'
  ));
