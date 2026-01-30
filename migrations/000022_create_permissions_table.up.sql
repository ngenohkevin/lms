-- Create permissions table for RBAC system
CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    code VARCHAR(100) UNIQUE NOT NULL,    -- e.g., "books.create"
    name VARCHAR(100) NOT NULL,           -- "Create Books"
    description TEXT,
    category VARCHAR(50) NOT NULL,        -- "books", "students", "reports"
    is_system BOOLEAN DEFAULT false,      -- System permissions cannot be deleted
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for efficient lookups
CREATE INDEX idx_permissions_code ON permissions(code);
CREATE INDEX idx_permissions_category ON permissions(category);

-- Seed default permissions
INSERT INTO permissions (code, name, description, category, is_system) VALUES
    -- Books permissions
    ('books.view', 'View Books', 'View book listings and details', 'books', true),
    ('books.create', 'Create Books', 'Add new books to the library', 'books', true),
    ('books.update', 'Update Books', 'Edit existing book information', 'books', true),
    ('books.delete', 'Delete Books', 'Remove books from the library', 'books', true),

    -- Students permissions
    ('students.view', 'View Students', 'View student listings and details', 'students', true),
    ('students.create', 'Create Students', 'Add new students to the system', 'students', true),
    ('students.update', 'Update Students', 'Edit existing student information', 'students', true),
    ('students.delete', 'Delete Students', 'Remove students from the system', 'students', true),

    -- Transactions permissions
    ('transactions.view', 'View Transactions', 'View borrowing and return transactions', 'transactions', true),
    ('transactions.borrow', 'Borrow Books', 'Process book borrowing for students', 'transactions', true),
    ('transactions.return', 'Return Books', 'Process book returns', 'transactions', true),

    -- Reservations permissions
    ('reservations.view', 'View Reservations', 'View book reservations', 'reservations', true),
    ('reservations.manage', 'Manage Reservations', 'Create, cancel, and fulfill reservations', 'reservations', true),

    -- Reports permissions
    ('reports.view', 'View Reports', 'Access library reports and analytics', 'reports', true),
    ('reports.export', 'Export Reports', 'Export reports to various formats', 'reports', true),

    -- Users permissions
    ('users.view', 'View Users', 'View user listings and details', 'users', true),
    ('users.manage', 'Manage Users', 'Create, edit, and deactivate users', 'users', true),

    -- Invites permissions
    ('invites.manage', 'Manage Invites', 'Send and manage user invitations', 'invites', true),

    -- Permissions permissions
    ('permissions.view', 'View Permissions', 'View role and user permissions', 'permissions', true),
    ('permissions.manage', 'Manage Permissions', 'Edit role permissions and user overrides', 'permissions', true),

    -- Fines permissions
    ('fines.view', 'View Fines', 'View fine information', 'fines', true),
    ('fines.manage', 'Manage Fines', 'Waive, adjust, and manage fines', 'fines', true),

    -- Notifications permissions
    ('notifications.send', 'Send Notifications', 'Send notifications to students', 'notifications', true),

    -- Categories permissions
    ('categories.manage', 'Manage Categories', 'Create, edit, and delete book categories', 'categories', true);
