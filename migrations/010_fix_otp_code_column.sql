-- Migration 010: Fix otp_codes.code column length
--
-- Bug: code was VARCHAR(10), but HashOTP() stores a SHA-256 hex digest
-- which is always 64 characters. Every INSERT into otp_codes was failing
-- silently, so OTP emails / WhatsApp messages were never sent.
--
ALTER TABLE otp_codes ALTER COLUMN code TYPE TEXT;
