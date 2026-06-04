import { ENV_CONFIG } from './app/config';

const isDevelopment = process.env.NODE_ENV === 'development';

// Create ENV as a getter-based object to ensure runtime evaluation
let cachedEnv: any = null;

function getEnv() {
  if (cachedEnv) return cachedEnv;
  
  cachedEnv = {
    NODE_ENV: process.env.NODE_ENV,
    PRODUCTION: process.env.PRODUCTION ?? !isDevelopment,
    SOURCEMAP: process.env.SOURCEMAP,
    KAI_TESTING: process.env.KAI_TESTING,
    get ORIGIN() { return ENV_CONFIG.origin; },
    get ASSETS_HOST() { return ENV_CONFIG.assetsHost; },
    get API_EDP() { return ENV_CONFIG.apiUrl; },
    get V2_API_EDP() { return ENV_CONFIG.v2ApiUrl; },
    SENTRY_ENABLED: process.env.SENTRY_ENABLED || 'false',
    SENTRY_URL: process.env.SENTRY_URL,
    CAPTCHA_ENABLED: process.env.CAPTCHA_ENABLED || 'false',
    CAPTCHA_SITE_KEY: process.env.CAPTCHA_SITE_KEY,
    MINIO_ENDPOINT: process.env.MINIO_ENDPOINT,
    MINIO_POST: process.env.MINIO_PORT,
    MINIO_USE_SSL: process.env.MINIO_USE_SSL,
    MINIO_ACCESS_KEY: process.env.MINIO_ACCESS_KEY,
    MINIO_SECRET_KEY: process.env.MINIO_SECRET_KEY,
    VERSION: process.env.VERSION || '1.25.0',
    TRACKER_ENABLED: process.env.TRACKER_ENABLED || 'false',
    TRACKER_VERSION: process.env.TRACKER_VERSION || '17.1.6',
    TRACKER_MAJOR_VERSION: process.env.TRACKER_MAJOR_VERSION || '17',
    TRACKER_PROJECT_KEY: process.env.TRACKER_PROJECT_KEY,
    COMMIT_HASH: process.env.COMMIT_HASH,
    TRACKER_HOST: process.env.TRACKER_HOST,
    TEST_FOSS_LOGIN: !isDevelopment ? undefined : process.env.TEST_FOSS_LOGIN,
    TEST_FOSS_PASSWORD: !isDevelopment
      ? undefined
      : process.env.TEST_FOSS_PASSWORD,
    STRIPE_KEY: process.env.STRIPE_KEY,
    CRISP_KEY: process.env.CRISP_KEY,
  };
  
  return cachedEnv;
}

const ENV = getEnv();

export default ENV;
