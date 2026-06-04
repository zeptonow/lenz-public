import os
import sys

# Set the ENV_FILE before importing decouple
os.environ['ENV_FILE'] = '.env.s1'

from or_config import config

print(f"ENV_FILE: {os.environ.get('ENV_FILE')}")
print(f"EXP_SESSIONS_SEARCH: {config('EXP_SESSIONS_SEARCH', cast=bool, default=False)}")
print(f"EXP_METRICS: {config('EXP_METRICS', cast=bool, default=False)}")
print(f"CH_ENABLED: {config('CH_ENABLED', cast=bool, default=True)}")
