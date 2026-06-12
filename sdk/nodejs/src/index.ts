/**
 * Hystersis - Node.js/Browser SDK
 * 
 * Persistent memory infrastructure for AI agents.
 * Memory that adapts. Intelligence that compounds.
 */

import { HystersisClient } from './client';
import { 
  HystersisError, AuthenticationError, NotFoundError, 
  ValidationError, RateLimitError, ServerError 
} from './errors';

const Hystersis = HystersisClient;

export { HystersisClient, Hystersis };
export { HystersisError, AuthenticationError, NotFoundError, ValidationError, RateLimitError, ServerError };
export type * from './types';

export default HystersisClient;
