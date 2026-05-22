/**
 * Error Classes
 */

export class HystersisError extends Error {
  constructor(
    message: string,
    public code?: string,
    public statusCode?: number,
    public response?: Response
  ) {
    super(message);
    this.name = 'HystersisError';
  }
}

export class AuthenticationError extends HystersisError {
  constructor(message: string, statusCode?: number, response?: Response) {
    super(message, 'AUTHENTICATION_ERROR', statusCode, response);
    this.name = 'AuthenticationError';
  }
}

export class NotFoundError extends HystersisError {
  constructor(message: string, statusCode?: number, response?: Response) {
    super(message, 'NOT_FOUND', statusCode, response);
    this.name = 'NotFoundError';
  }
}

export class ValidationError extends HystersisError {
  constructor(message: string, statusCode?: number, response?: Response) {
    super(message, 'VALIDATION_ERROR', statusCode, response);
    this.name = 'ValidationError';
  }
}

export class RateLimitError extends HystersisError {
  constructor(message: string, statusCode?: number, response?: Response) {
    super(message, 'RATE_LIMIT', statusCode, response);
    this.name = 'RateLimitError';
  }
}

export class ServerError extends HystersisError {
  constructor(message: string, statusCode?: number, response?: Response) {
    super(message, 'SERVER_ERROR', statusCode, response);
    this.name = 'ServerError';
  }
}