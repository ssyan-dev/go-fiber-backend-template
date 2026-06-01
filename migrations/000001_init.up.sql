CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE user_role AS ENUM('default', 'admin');

CREATE TYPE auth_provider AS ENUM('google', 'github', 'yandex');

CREATE TABLE users (
	id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
	email VARCHAR(255) UNIQUE NOT NULL,
	role user_role NOT NULL DEFAULT 'default',
	avatar_url TEXT,
	password_hash VARCHAR(255),
	is_banned BOOLEAN DEFAULT FALSE,
	is_email_verified BOOLEAN DEFAULT FALSE,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
	updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_users_email ON users (email);

CREATE TABLE user_oauth_providers (
	id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
	user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	provider auth_provider NOT NULL,
	provider_user_id VARCHAR(255) NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
	UNIQUE (provider, provider_user_id),
	UNIQUE (user_id, provider)
);

CREATE INDEX idx_oauth_provider_user ON user_oauth_providers (user_id);
