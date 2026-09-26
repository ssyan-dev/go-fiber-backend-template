CREATE TYPE verification_type AS ENUM('email_verification', 'password_reset');

CREATE TABLE verification_codes (
	id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
	user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	code TEXT NOT NULL,
	type verification_type NOT NULL,
	expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_verification_codes_user_id ON verification_codes (user_id);
