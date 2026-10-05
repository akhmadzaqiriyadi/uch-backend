-- 000008_create_events_and_registrations.down.sql

DROP TABLE IF EXISTS event_registrations CASCADE;
DROP TABLE IF EXISTS events CASCADE;

DELETE FROM role_permissions WHERE permission_id IN (
    'events:read', 'events:create', 'events:update', 'events:delete', 'events:manage'
);

DELETE FROM permissions WHERE id IN (
    'events:read', 'events:create', 'events:update', 'events:delete', 'events:manage'
);
