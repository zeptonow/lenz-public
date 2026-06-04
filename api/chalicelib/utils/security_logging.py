import logging
import re

class JWTRedactionFilter(logging.Filter):
    """
    Redacts JWT tokens and other sensitive auth patterns from log records.
    
    Patterns redacted:
    - JWT tokens (three Base64url segments separated by dots)
    - Authorization Bearer tokens
    - Keys containing 'secret', 'password', 'token', 'jwt' in request/response bodies
    - Sensitive query parameters
    """
    
    # Pattern for JWT tokens: three Base64url segments (roughly)
    JWT_PATTERN = re.compile(
        r'(eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)',
        re.IGNORECASE
    )
    
    # Pattern for Authorization Bearer tokens
    BEARER_PATTERN = re.compile(
        r'(Bearer\s+)[A-Za-z0-9_\-\.]+',
        re.IGNORECASE
    )
    
    # Pattern for common auth parameters in URLs and bodies
    AUTH_PARAMS_PATTERN = re.compile(
        r'([?&])(jwt|token|access_token|authorization|api_key|apikey|secret|password)=([^&\s"\'<>]+)',
        re.IGNORECASE
    )
    
    # Pattern for JSON key-value pairs with sensitive keys
    JSON_SENSITIVE_PATTERN = re.compile(
        r'(["\']?)(jwt|token|access_token|refresh_token|api_key|apikey|secret|password)(["\']?\s*:\s*)(["\']?)([A-Za-z0-9_\-\.]+)(["\']?)',
        re.IGNORECASE
    )
    
    REDACTED = '[REDACTED]'
    
    def filter(self, record: logging.LogRecord) -> bool:
        """Apply redaction to the log record message and args."""
        if isinstance(record.msg, str):
            record.msg = self.redact(record.msg)
        
        if record.args:
            if isinstance(record.args, dict):
                for key in record.args:
                    value = record.args[key]
                    if isinstance(value, str):
                        record.args[key] = self.redact(value)
            elif isinstance(record.args, (tuple, list)):
                record.args = tuple(
                    self.redact(arg) if isinstance(arg, str) else arg
                    for arg in record.args
                )
        
        return True
    
    @classmethod
    def redact(cls, text: str) -> str:
        """Redact sensitive patterns from text."""
        if not isinstance(text, str):
            return text
        
        text = cls.JWT_PATTERN.sub(cls.REDACTED, text)
        text = cls.BEARER_PATTERN.sub(r'\1' + cls.REDACTED, text)
        text = cls.AUTH_PARAMS_PATTERN.sub(r'\1\2=' + cls.REDACTED, text)
        text = cls.JSON_SENSITIVE_PATTERN.sub(
            r'\1\2\3\4' + cls.REDACTED + r'\6',
            text
        )
        
        return text
