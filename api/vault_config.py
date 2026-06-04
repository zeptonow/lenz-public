import json
import os
from typing import Any, Dict


class VaultConfig:
    """
    Config loader that reads from Vault JSON files when VAULT_ENABLED=true,
    similar to how the Go services use Viper.
    """
    
    VAULT_STATIC_FILE = "/vault/secrets/static.json"
    VAULT_DYNAMIC_FILE = "/vault/secrets/dynamic.json"
    
    def __init__(self):
        self._config: Dict[str, Any] = {}
        self._load_config()
    
    def _load_config(self):
        """Load configuration from Vault JSON files or fallback to environment variables"""
        vault_enabled = os.getenv("VAULT_ENABLED", "false").lower() in ("true", "1", "yes")
        
        if vault_enabled:
            print(f"Loading config from Vault at {self.VAULT_STATIC_FILE}")
            self._load_json_file(self.VAULT_STATIC_FILE)
            
            # Try to load dynamic config (might not exist)
            if os.path.exists(self.VAULT_DYNAMIC_FILE):
                print(f"Loading dynamic config from {self.VAULT_DYNAMIC_FILE}")
                self._load_json_file(self.VAULT_DYNAMIC_FILE)
            else:
                print("Dynamic config file not found, skipping")
        else:
            print("VAULT_ENABLED=false, using .env file or environment variables")
    
    def _load_json_file(self, file_path: str):
        """Load and merge JSON config file"""
        try:
            with open(file_path, 'r') as f:
                data = json.load(f)
                self._config.update(data)
                print(f"Loaded {len(data)} config values from {file_path}")
        except FileNotFoundError:
            print(f"Config file not found: {file_path}")
        except json.JSONDecodeError as e:
            print(f"Error parsing JSON from {file_path}: {e}")
            raise
    
    def get(self, key: str, default: Any = None, cast: type = str) -> Any:
        """
        Get config value, checking Vault config first, then environment variables,
        then returning default.
        
        Compatible with decouple's config() interface.
        """
        # Priority: Vault JSON > Environment Variable > Default
        value = self._config.get(key) or os.getenv(key)
        
        if value is None:
            return default
        
        # Cast to requested type
        if cast == bool:
            if isinstance(value, bool):
                return value
            return str(value).lower() in ('true', '1', 'yes')
        elif cast == int:
            return int(value)
        elif cast == float:
            return float(value)
        else:
            return str(value)
    
    def __call__(self, key: str, default: Any = None, cast: type = str) -> Any:
        """Make VaultConfig callable like decouple's config()"""
        return self.get(key, default, cast)


# Global singleton instance
_vault_config = None


def get_config():
    """Get or create the global VaultConfig instance"""
    global _vault_config
    if _vault_config is None:
        _vault_config = VaultConfig()
    return _vault_config
