-- Add is_active flag for soft-disable of accounts
ALTER TABLE users ADD COLUMN is_active BOOLEAN DEFAULT TRUE NOT NULL;

-- Add deleted_at for soft delete support
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;

-- Index for soft-delete queries (exclude soft-deleted users)
CREATE INDEX idx_users_active ON users (is_active) WHERE deleted_at IS NULL;

-- Index for role-based queries
CREATE INDEX idx_users_role ON users (role);

-- Index for admin listing with status
CREATE INDEX idx_users_role_active ON users (role, is_active) WHERE deleted_at IS NULL;
