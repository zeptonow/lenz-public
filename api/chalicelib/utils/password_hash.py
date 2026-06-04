"""Password hashing utility using bcrypt (replacement for pgcrypto)"""
import bcrypt


def hash_password(password: str) -> str:
    """Hash a password using bcrypt (compatible with pgcrypto's crypt())"""
    # Encode the password to bytes and hash it
    password_bytes = password.encode('utf-8')
    # Generate a salt and hash the password
    salt = bcrypt.gensalt(rounds=12)  # Same as gen_salt('bf', 12) in pgcrypto
    hashed = bcrypt.hashpw(password_bytes, salt)
    # Return as string
    return hashed.decode('utf-8')


def verify_password(plain_password: str, hashed_password: str) -> bool:
    """Verify a password against its bcrypt hash"""
    try:
        password_bytes = plain_password.encode('utf-8')
        hashed_bytes = hashed_password.encode('utf-8')
        return bcrypt.checkpw(password_bytes, hashed_bytes)
    except Exception:
        return False
