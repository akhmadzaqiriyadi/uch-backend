-- Create roles table
CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create permissions table
CREATE TABLE IF NOT EXISTS permissions (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    module VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create role_permissions junction table
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id VARCHAR(50) REFERENCES roles(id) ON DELETE CASCADE,
    permission_id VARCHAR(100) REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Insert Default Roles
INSERT INTO roles (id, name, description) VALUES
('admin', 'Administrator', 'Full system access and administrative control'),
('manager', 'Manager', 'Elevated access for management and user oversight'),
('user', 'Standard User', 'Regular account with standard user access')
ON CONFLICT (id) DO NOTHING;

-- Insert Granular Permissions
INSERT INTO permissions (id, name, module, description) VALUES
('users:read', 'View Users', 'users', 'Ability to list and view user details'),
('users:create', 'Create User', 'users', 'Ability to create new users directly'),
('users:update', 'Update User', 'users', 'Ability to update user records'),
('users:delete', 'Delete User', 'users', 'Ability to delete users from the system'),
('roles:read', 'View Roles', 'roles', 'Ability to view system roles and permissions'),
('roles:manage', 'Manage Roles', 'roles', 'Ability to assign or modify roles and permissions'),
('uploads:create', 'Upload Files', 'uploads', 'Ability to upload files and media'),
('audit:read', 'View Audit Logs', 'audit', 'Ability to view system audit logs')
ON CONFLICT (id) DO NOTHING;

-- Assign Permissions to Roles
-- Admin gets ALL permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 'admin', id FROM permissions
ON CONFLICT DO NOTHING;

-- Manager gets users:read, users:update, uploads:create, audit:read
INSERT INTO role_permissions (role_id, permission_id) VALUES
('manager', 'users:read'),
('manager', 'users:update'),
('manager', 'uploads:create'),
('manager', 'audit:read')
ON CONFLICT DO NOTHING;

-- User gets uploads:create
INSERT INTO role_permissions (role_id, permission_id) VALUES
('user', 'uploads:create')
ON CONFLICT DO NOTHING;
