import asyncio
import logging
import os
from typing import List, Optional
from google.cloud import vision
from google.cloud import storage
import requests
from PIL import Image
import io
import re

logger = logging.getLogger(__name__)

class OCRService:
    def __init__(self):
        """Initialize Google Cloud Vision API client and Storage client"""
        try:
            self.client = vision.ImageAnnotatorClient()
            self.storage_client = storage.Client()
            logger.info("Google Cloud Vision API client initialized successfully")
        except Exception as e:
            logger.error(f"Failed to initialize Vision API client: {e}")
            raise

    async def extract_text_from_images(self, image_urls: List[str]) -> List[str]:
        """Extract text from multiple recipe images"""
        extracted_texts = []

        for i, url in enumerate(image_urls):
            try:
                text = await self._extract_text_from_url(url)
                if text.strip():
                    extracted_texts.append(text)
                    logger.info(f"Successfully extracted text from image {i+1}/{len(image_urls)}")
                else:
                    logger.warning(f"No text found in image {i+1}/{len(image_urls)}")
            except Exception as e:
                logger.error(f"Failed to extract text from image {i+1}: {e}")
                # Continue processing other images even if one fails
                continue

        return extracted_texts

    async def _extract_text_from_url(self, image_url: str) -> str:
        """Extract text from a single image URL with comprehensive error handling"""
        try:
            # Try to download the image with fallback mechanisms
            image_content_raw = await self._download_image_with_fallback(image_url)

            # Preprocess image for better OCR results
            image_content = await self._preprocess_image(image_content_raw)

            # Perform OCR with retry logic and enhanced error handling
            detected_text = await self._perform_ocr_with_retry(image_content, image_url)
            return detected_text

        except Exception as e:
            logger.error(f"OCR processing failed for URL {image_url}: {e}")
            raise

    async def _perform_ocr_with_retry(self, image_content: bytes, image_url: str) -> str:
        """Perform OCR with retry logic and enhanced error handling"""
        max_retries = 3
        base_delay = 1.0

        for attempt in range(1, max_retries + 1):
            try:
                logger.info(f"OCR attempt {attempt}/{max_retries} for image: {image_url}")

                # Create Vision API image object
                image = vision.Image(content=image_content)

                # Perform text detection with timeout considerations
                response = self.client.text_detection(image=image)

                # Enhanced error handling for Vision API response
                if response.error and response.error.message:
                    error_code = getattr(response.error, 'code', 'UNKNOWN')
                    error_msg = response.error.message

                    # Classify errors and determine if retry is worthwhile
                    if self._is_retryable_vision_error(error_code, error_msg):
                        if attempt < max_retries:
                            delay = base_delay * (2 ** (attempt - 1))  # Exponential backoff
                            logger.warning(f"Retryable Vision API error (attempt {attempt}): {error_msg}. Retrying in {delay}s...")
                            await asyncio.sleep(delay)
                            continue
                        else:
                            raise Exception(f"Vision API error after {max_retries} attempts: {error_msg}")
                    else:
                        raise Exception(f"Non-retryable Vision API error: {error_msg}")

                # Extract text annotations
                texts = response.text_annotations
                if texts and len(texts) > 0:
                    # The first annotation contains the entire detected text
                    detected_text = texts[0].description
                    if detected_text:
                        logger.info(f"Successfully extracted {len(detected_text)} characters of text")
                        return detected_text.strip()
                    else:
                        logger.info("Vision API returned empty text description")
                        return ""
                else:
                    logger.info("No text annotations found in image")
                    return ""

            except Exception as e:
                error_type = type(e).__name__
                if attempt < max_retries and self._is_retryable_exception(e):
                    delay = base_delay * (2 ** (attempt - 1))
                    logger.warning(f"OCR failed with {error_type} (attempt {attempt}): {e}. Retrying in {delay}s...")
                    await asyncio.sleep(delay)
                    continue
                else:
                    logger.error(f"OCR failed permanently with {error_type}: {e}")
                    raise

        # This should not be reached, but just in case
        raise Exception(f"OCR failed after {max_retries} attempts")

    def _is_retryable_vision_error(self, error_code: str, error_message: str) -> bool:
        """Determine if a Vision API error is worth retrying"""
        # Retryable error codes and patterns
        retryable_codes = [
            'RATE_LIMIT_EXCEEDED',
            'QUOTA_EXCEEDED',
            'INTERNAL',
            'UNAVAILABLE',
            'DEADLINE_EXCEEDED'
        ]

        retryable_messages = [
            'rate limit',
            'quota exceeded',
            'internal error',
            'service unavailable',
            'timeout',
            'deadline exceeded'
        ]

        # Check error code
        if error_code in retryable_codes:
            return True

        # Check error message patterns
        error_lower = error_message.lower()
        for pattern in retryable_messages:
            if pattern in error_lower:
                return True

        return False

    def _is_retryable_exception(self, exception: Exception) -> bool:
        """Determine if a general exception is worth retrying"""
        retryable_types = [
            'ConnectionError',
            'Timeout',
            'ServiceUnavailable',
            'InternalServerError'
        ]

        exception_type = type(exception).__name__
        return exception_type in retryable_types

    async def _preprocess_image(self, image_content: bytes) -> bytes:
        """Preprocess image to improve OCR accuracy"""
        try:
            # Open image with PIL
            image = Image.open(io.BytesIO(image_content))

            # Convert to RGB if needed
            if image.mode != 'RGB':
                image = image.convert('RGB')

            # Auto-rotate based on EXIF data
            image = self._auto_rotate(image)

            # Enhance contrast slightly for better text recognition
            from PIL import ImageEnhance
            enhancer = ImageEnhance.Contrast(image)
            image = enhancer.enhance(1.2)

            # Convert back to bytes
            output_buffer = io.BytesIO()
            image.save(output_buffer, format='JPEG', quality=95)
            return output_buffer.getvalue()

        except Exception as e:
            logger.warning(f"Image preprocessing failed, using original: {e}")
            return image_content

    def _auto_rotate(self, image: Image.Image) -> Image.Image:
        """Auto-rotate image based on EXIF orientation data"""
        try:
            # Get EXIF data
            exif = image._getexif()
            if exif is not None:
                orientation = exif.get(274)  # 274 is the orientation tag

                if orientation == 3:
                    image = image.rotate(180, expand=True)
                elif orientation == 6:
                    image = image.rotate(270, expand=True)
                elif orientation == 8:
                    image = image.rotate(90, expand=True)

            return image
        except Exception:
            # If EXIF processing fails, return original image
            return image

    def _is_gcs_url(self, url: str) -> bool:
        """Check if URL is a Google Cloud Storage URL"""
        return url.startswith('https://storage.googleapis.com/')

    async def _download_image_with_fallback(self, image_url: str) -> bytes:
        """Download image with comprehensive error handling and fallback mechanisms"""
        download_errors = []

        # Primary method: GCS client (for GCS URLs)
        if self._is_gcs_url(image_url):
            try:
                logger.info(f"Attempting GCS client download for: {image_url}")
                return await self._download_from_gcs(image_url)
            except Exception as gcs_error:
                logger.warning(f"GCS client download failed: {gcs_error}")
                download_errors.append(f"GCS client: {str(gcs_error)}")

                # Fallback: Try HTTP download even for GCS URLs
                logger.info(f"Falling back to HTTP download for GCS URL: {image_url}")
                try:
                    return await self._download_via_http(image_url)
                except Exception as http_error:
                    logger.warning(f"HTTP fallback also failed: {http_error}")
                    download_errors.append(f"HTTP fallback: {str(http_error)}")
        else:
            # Non-GCS URL: Use HTTP download
            try:
                logger.info(f"Attempting HTTP download for: {image_url}")
                return await self._download_via_http(image_url)
            except Exception as http_error:
                logger.warning(f"HTTP download failed: {http_error}")
                download_errors.append(f"HTTP: {str(http_error)}")

        # If all methods failed, raise comprehensive error
        error_summary = "; ".join(download_errors)
        raise Exception(f"All download methods failed for {image_url}. Errors: {error_summary}")

    async def _download_from_gcs(self, gcs_url: str) -> bytes:
        """Download blob from Google Cloud Storage with enhanced error handling"""
        try:
            # Parse GCS URL: https://storage.googleapis.com/bucket/path/to/object
            # Extract bucket name and object path
            url_pattern = r'https://storage\.googleapis\.com/([^/]+)/(.+)'
            match = re.match(url_pattern, gcs_url)

            if not match:
                raise ValueError(f"Invalid GCS URL format: {gcs_url}")

            bucket_name = match.group(1)
            object_path = match.group(2)

            # Validate bucket name to prevent injection
            if not bucket_name or '/' in bucket_name or bucket_name.startswith('.'):
                raise ValueError(f"Invalid bucket name: {bucket_name}")

            # Get bucket and blob with timeout
            bucket = self.storage_client.bucket(bucket_name)
            blob = bucket.blob(object_path)

            # Download blob content with timeout and retry logic
            logger.info(f"Downloading blob from GCS: {bucket_name}/{object_path}")

            # Check if blob exists first
            if not blob.exists():
                raise FileNotFoundError(f"Blob does not exist: {bucket_name}/{object_path}")

            content = blob.download_as_bytes()

            if not content:
                raise ValueError(f"Downloaded empty content from GCS: {gcs_url}")

            logger.info(f"Successfully downloaded {len(content)} bytes from GCS")
            return content

        except Exception as e:
            # Classify error types for better debugging
            error_type = type(e).__name__
            logger.error(f"GCS download failed ({error_type}): {e}")
            raise

    async def _download_via_http(self, image_url: str) -> bytes:
        """Download image via HTTP with enhanced error handling and retries"""
        max_retries = 3
        timeout = 30

        for attempt in range(1, max_retries + 1):
            try:
                logger.info(f"HTTP download attempt {attempt}/{max_retries} for: {image_url}")

                # Set proper headers to mimic browser request
                headers = {
                    'User-Agent': 'Mozilla/5.0 (compatible; DigitalRecipesBot/1.0)',
                    'Accept': 'image/*,*/*;q=0.8',
                    'Accept-Encoding': 'gzip, deflate',
                    'Connection': 'keep-alive'
                }

                response = requests.get(
                    image_url,
                    timeout=timeout,
                    headers=headers,
                    stream=True,  # Stream large files
                    allow_redirects=True
                )
                response.raise_for_status()

                # Validate content type
                content_type = response.headers.get('Content-Type', '').lower()
                if not content_type.startswith('image/'):
                    logger.warning(f"Unexpected content type: {content_type} for URL: {image_url}")

                # Download content with size limit
                max_size = 50 * 1024 * 1024  # 50MB limit
                content = b''
                for chunk in response.iter_content(chunk_size=8192):
                    content += chunk
                    if len(content) > max_size:
                        raise ValueError(f"Image too large: {len(content)} bytes exceeds {max_size} bytes")

                if not content:
                    raise ValueError(f"Downloaded empty content from HTTP: {image_url}")

                logger.info(f"Successfully downloaded {len(content)} bytes via HTTP")
                return content

            except requests.exceptions.RequestException as e:
                error_msg = f"HTTP request failed on attempt {attempt}: {e}"
                if attempt == max_retries:
                    logger.error(error_msg)
                    raise Exception(error_msg)
                else:
                    logger.warning(f"{error_msg}, retrying...")
                    # Brief delay before retry
                    await asyncio.sleep(1)
            except Exception as e:
                error_msg = f"HTTP download failed on attempt {attempt}: {e}"
                logger.error(error_msg)
                raise Exception(error_msg)