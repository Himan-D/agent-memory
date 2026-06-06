/**
 * Hystersis - Node.js/Browser SDK
 * 
 * Persistent memory infrastructure for AI agents.
 * Memory that adapts. Intelligence that compounds.
 */

import { HystersisClient } from './client';
import type { 
  HystersisConfig, RetryConfig, RateLimitConfig, TimeoutConfig, 
  RequestInterceptor, ResponseInterceptor 
} from './types';
import { 
  HystersisError, AuthenticationError, NotFoundError, 
  ValidationError, RateLimitError, ServerError 
} from './errors';

export { HystersisClient };
/** Alias matching Python SDK and documentation examples */
export { HystersisClient as Hystersis };
export { HystersisError, AuthenticationError, NotFoundError, ValidationError, RateLimitError, ServerError };
export type { 
  HystersisConfig, RetryConfig, RateLimitConfig, TimeoutConfig, 
  RequestInterceptor, ResponseInterceptor 
};

export default HystersisClient;