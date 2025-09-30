'use client';

import DOMPurify from 'dompurify';
import { useMemo } from 'react';

interface SafeContentProps {
  content: string | null | undefined;
  className?: string;
  allowedTags?: string[];
  preserveWhitespace?: boolean;
}

/**
 * SafeContent component provides XSS protection by sanitizing user-generated content
 * before rendering it in the DOM.
 */
export default function SafeContent({
  content,
  className = '',
  allowedTags = [],
  preserveWhitespace = true
}: SafeContentProps) {
  const sanitizedContent = useMemo(() => {
    if (!content) return '';

    // Configure DOMPurify to strip all HTML by default for maximum security
    const config = {
      ALLOWED_TAGS: allowedTags.length > 0 ? allowedTags : [], // No HTML tags by default
      ALLOWED_ATTR: [], // No attributes allowed
      RETURN_DOM: false,
      RETURN_DOM_FRAGMENT: false,
      RETURN_DOM_IMPORT: false,
      SANITIZE_DOM: true
    };

    return DOMPurify.sanitize(content.trim(), config);
  }, [content, allowedTags]);

  if (!sanitizedContent) {
    return null;
  }

  const finalClassName = preserveWhitespace
    ? `whitespace-pre-line ${className}`.trim()
    : className;

  // For inline content, return sanitized text directly without div wrapper
  if (className.includes('inline')) {
    return <>{sanitizedContent}</>;
  }

  return (
    <div
      className={finalClassName}
      // Use dangerouslySetInnerHTML only after sanitization
      dangerouslySetInnerHTML={{ __html: sanitizedContent }}
    />
  );
}