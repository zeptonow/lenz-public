// Frontend configuration - environment-specific settings
// These are public URLs, not secrets, so they're safe to commit

interface EnvironmentConfig {
  apiUrl: string;
  v2ApiUrl: string;
  assetsHost: string;
  origin: string;
}

const ENVIRONMENTS: Record<string, EnvironmentConfig> = {
  local: {
    origin: 'http://localhost:8000',
    apiUrl: 'http://localhost:8000/api',
    v2ApiUrl: 'http://localhost:8000',
    assetsHost: 'http://localhost:8000',
  },
  qa: {
    origin: 'http://localhost:8000',
    apiUrl: 'http://localhost:8000',
    v2ApiUrl: 'http://localhost:8000',
    assetsHost: 'http://localhost:8000',
  },
  prod: {
    origin: 'http://localhost:8000',
    apiUrl: 'http://localhost:8000/api',
    v2ApiUrl: 'http://localhost:8000',
    assetsHost: 'http://localhost:8000',
  },
};

// Detect environment from hostname or NODE_ENV
function detectEnvironment(): string {
  // Check for manual override first (useful for local development)
  if (typeof window !== 'undefined') {
    const override = localStorage.getItem('__env_override');
    if (override && ENVIRONMENTS[override]) {
      console.log(`[Config] Using environment override: ${override}`);
      return override;
    }
  }
  
  // In browser, detect from hostname
  if (typeof window !== 'undefined') {
    const hostname = window.location.hostname;
    
    if (hostname === 'localhost' || hostname === '127.0.0.1') {
      return 'local';
    }
    
    if (hostname.includes('http://localhost:8000')) {
      return 'qa';
    }
    
    if (hostname.includes('http://localhost:8000')) {
      return 'prod';
    }
    
    // Default to prod for unknown domains
    return 'prod';
  }
  
  // During build/SSR, default to prod (will be corrected on client-side)
  return 'prod';
}

// Use a getter to ensure detection happens at runtime, not build time
let cachedEnv: string | null = null;
let cachedConfig: EnvironmentConfig | null = null;

function getCurrentEnvironment(): string {
  if (cachedEnv === null) {
    cachedEnv = detectEnvironment();
    console.log(`[Config] Detected environment: ${cachedEnv}`);
  }
  return cachedEnv;
}

function getConfig(): EnvironmentConfig {
  if (cachedConfig === null) {
    const env = getCurrentEnvironment();
    cachedConfig = ENVIRONMENTS[env];
    console.log(`[Config] Using config for ${env}:`, cachedConfig);
  }
  return cachedConfig;
}

// Export as getter property to ensure runtime detection
export const ENV_CONFIG = new Proxy({} as EnvironmentConfig, {
  get(target, prop) {
    const config = getConfig();
    return config[prop as keyof EnvironmentConfig];
  }
});

// Also export for debugging - use getter to ensure runtime evaluation
export function getCurrentEnv(): string {
  return getCurrentEnvironment();
}
