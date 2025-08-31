// Upload configuration constants
export const UPLOAD_CONFIG = {
  // File size limits
  MAX_FILE_SIZE_MB: 10,
  MAX_FILE_SIZE_BYTES: 10 * 1024 * 1024,
  
  // File type restrictions
  ALLOWED_TYPES: ['image/jpeg', 'image/png', 'image/webp'] as const,
  ALLOWED_MIME_TYPES: ['image/jpeg', 'image/png', 'image/webp'] as const,
  
  // Upload limits
  MAX_IMAGES: 5,
  
  // Security constants
  MAX_HEADER_VALUE_LENGTH: 1000,
  ALLOWED_UPLOAD_HEADERS: [
    'Content-Type',
    'Cache-Control',
    'Content-Length-Range'
  ] as const,
  METADATA_HEADER_PREFIX: 'x-goog-meta-',
  
  // Progress and UI
  PROGRESS_THROTTLE_MS: 100, // Throttle progress updates
  RETRY_ATTEMPTS: 3,
  RETRY_DELAY_MS: 1000
} as const;

// Validation functions
export const validateFileType = (file: File): boolean => {
  return UPLOAD_CONFIG.ALLOWED_MIME_TYPES.includes(file.type as any);
};

export const validateFileSize = (file: File): boolean => {
  return file.size <= UPLOAD_CONFIG.MAX_FILE_SIZE_BYTES;
};

export const getFileTypeError = (): string => {
  return `Please select only image files (${UPLOAD_CONFIG.ALLOWED_TYPES.join(', ')})`;
};

export const getFileSizeError = (): string => {
  return `File size must be less than ${UPLOAD_CONFIG.MAX_FILE_SIZE_MB}MB`;
};