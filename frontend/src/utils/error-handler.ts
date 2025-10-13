import { AxiosError } from 'axios';

export interface ErrorInfo {
  message: string;
  code?: string;
  retryable: boolean;
  userAction?: string;
  details?: any;
}

export class APIError extends Error {
  public errorInfo: ErrorInfo;
  public statusCode?: number;

  constructor(message: string, errorInfo: ErrorInfo, statusCode?: number) {
    super(message);
    this.name = 'APIError';
    this.errorInfo = errorInfo;
    this.statusCode = statusCode;
  }
}

/**
 * Maps API errors to user-friendly messages
 */
export function getErrorMessage(error: unknown): ErrorInfo {
  // Handle axios errors
  if (error instanceof AxiosError) {
    const status = error.response?.status;
    const data = error.response?.data;

    // Network errors
    if (!error.response) {
      return {
        message: 'Unable to connect to the server. Please check your internet connection.',
        code: 'NETWORK_ERROR',
        retryable: true,
        userAction: 'Check your connection and try again'
      };
    }

    // Handle specific HTTP status codes
    switch (status) {
      case 400:
        return {
          message: data?.error || 'Invalid request. Please check your input.',
          code: 'BAD_REQUEST',
          retryable: false,
          userAction: 'Review and correct your input'
        };

      case 401:
        return {
          message: 'Your session has expired. Please log in again.',
          code: 'UNAUTHORIZED',
          retryable: false,
          userAction: 'Log in to continue'
        };

      case 403:
        return {
          message: data?.error || 'You don\'t have permission to perform this action.',
          code: 'FORBIDDEN',
          retryable: false,
          userAction: 'Contact support if you believe this is an error'
        };

      case 404:
        return {
          message: 'The requested resource was not found.',
          code: 'NOT_FOUND',
          retryable: false,
          userAction: 'Go back or refresh the page'
        };

      case 409:
        return {
          message: data?.error || 'This action conflicts with existing data.',
          code: 'CONFLICT',
          retryable: false,
          userAction: 'Review for duplicate entries'
        };

      case 429:
        return {
          message: 'Too many requests. Please slow down.',
          code: 'RATE_LIMITED',
          retryable: true,
          userAction: 'Wait a moment before trying again'
        };

      case 500:
      case 502:
      case 503:
      case 504:
        return {
          message: 'Server error. Our team has been notified.',
          code: 'SERVER_ERROR',
          retryable: true,
          userAction: 'Try again in a few moments'
        };

      default:
        return {
          message: data?.error || 'An unexpected error occurred.',
          code: 'UNKNOWN_ERROR',
          retryable: true,
          userAction: 'Try again or contact support'
        };
    }
  }

  // Handle abort errors (cancelled requests)
  if (error instanceof Error && error.name === 'AbortError') {
    return {
      message: 'Request was cancelled.',
      code: 'ABORTED',
      retryable: false
    };
  }

  // Handle generic errors
  if (error instanceof Error) {
    return {
      message: error.message || 'An unexpected error occurred.',
      code: 'GENERIC_ERROR',
      retryable: false
    };
  }

  // Fallback for unknown errors
  return {
    message: 'An unexpected error occurred.',
    code: 'UNKNOWN',
    retryable: false,
    details: error
  };
}

/**
 * Retry logic with exponential backoff
 */
export async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  maxRetries: number = 3,
  initialDelay: number = 1000
): Promise<T> {
  let lastError: unknown;

  for (let attempt = 0; attempt < maxRetries; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error;
      const errorInfo = getErrorMessage(error);

      // Don't retry if error is not retryable
      if (!errorInfo.retryable) {
        throw error;
      }

      // Don't retry on last attempt
      if (attempt === maxRetries - 1) {
        throw error;
      }

      // Exponential backoff
      const delay = initialDelay * Math.pow(2, attempt);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }

  throw lastError;
}

/**
 * Format validation errors for display
 */
export function formatValidationErrors(errors: Record<string, string>): string {
  const errorMessages = Object.entries(errors)
    .filter(([_, error]) => error)
    .map(([field, error]) => `${field}: ${error}`);

  if (errorMessages.length === 0) return '';
  if (errorMessages.length === 1) return errorMessages[0];

  return 'Please fix the following issues:\n' + errorMessages.join('\n');
}

/**
 * Log error for monitoring (in production, send to error tracking service)
 */
export function logError(error: unknown, context?: Record<string, any>): void {
  const errorInfo = getErrorMessage(error);

  // In production, send to error tracking service like Sentry
  if (process.env.NODE_ENV === 'production') {
    // TODO: Send to error tracking service
    console.error('Error logged:', {
      error: errorInfo,
      context,
      timestamp: new Date().toISOString()
    });
  } else {
    // In development, log to console with formatting
    console.group(`🚨 Error: ${errorInfo.code || 'UNKNOWN'}`);
    console.error('Message:', errorInfo.message);
    if (errorInfo.userAction) {
      console.log('User Action:', errorInfo.userAction);
    }
    if (context) {
      console.log('Context:', context);
    }
    console.error('Raw Error:', error);
    console.groupEnd();
  }
}