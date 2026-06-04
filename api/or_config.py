"""
Config module that provides a unified config() function.
Uses VaultConfig when VAULT_ENABLED=true (K8s), otherwise falls back to decouple.
"""
import os

# Check if we should use Vault config (case-insensitive)
vault_enabled = os.getenv("VAULT_ENABLED", "false").lower() in ("true", "1", "yes")

if vault_enabled:
    # Use VaultConfig for K8s deployments
    from vault_config import get_config
    config = get_config()
else:
    # Use decouple for local development
    from decouple import config

__all__ = ['config']
