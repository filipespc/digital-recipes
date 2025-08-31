'use client';

import React, { useState, useRef, DragEvent, ChangeEvent, useCallback } from 'react';
import { RecipeAPI } from '@/services/api';
import { UPLOAD_CONFIG, validateFileType, validateFileSize, getFileTypeError, getFileSizeError } from '@/constants/upload';

interface ImageFile {
  file: File;
  id: string;
  preview: string;
  progress: number;
  status: 'pending' | 'uploading' | 'completed' | 'error';
  error?: string;
}

interface ImageUploadProps {
  onUploadComplete?: (recipeId: number) => void;
  onError?: (error: string) => void;
  maxImages?: number;
}

export default function ImageUpload({ 
  onUploadComplete, 
  onError, 
  maxImages = UPLOAD_CONFIG.MAX_IMAGES 
}: ImageUploadProps) {
  const [images, setImages] = useState<ImageFile[]>([]);
  const [isDragOver, setIsDragOver] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const validateFile = (file: File): string | null => {
    // Check file type using configured validation
    if (!validateFileType(file)) {
      return getFileTypeError();
    }

    // Check file size using configured validation
    if (!validateFileSize(file)) {
      return getFileSizeError();
    }

    return null;
  };

  const createImageFile = (file: File): ImageFile => {
    // Use crypto.randomUUID() if available, fallback to secure alternative
    const generateSecureId = (): string => {
      if (typeof crypto !== 'undefined' && crypto.randomUUID) {
        return crypto.randomUUID();
      }
      // Fallback: generate cryptographically secure ID
      const array = new Uint8Array(16);
      crypto.getRandomValues(array);
      return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
    };

    return {
      file,
      id: generateSecureId(),
      preview: URL.createObjectURL(file),
      progress: 0,
      status: 'pending'
    };
  };

  const handleFiles = (files: FileList) => {
    const newImages: ImageFile[] = [];
    const errors: string[] = [];

    // Check if adding these files would exceed the limit
    if (images.length + files.length > maxImages) {
      errors.push(`Maximum ${maxImages} images allowed`);
      onError?.(errors[0]);
      return;
    }

    Array.from(files).forEach(file => {
      const error = validateFile(file);
      if (error) {
        errors.push(error);
      } else {
        newImages.push(createImageFile(file));
      }
    });

    if (errors.length > 0) {
      onError?.(errors[0]);
      return;
    }

    setImages(prev => [...prev, ...newImages]);
  };

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragOver(true);
  };

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragOver(false);
  };

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragOver(false);
    
    const files = e.dataTransfer.files;
    if (files.length > 0) {
      handleFiles(files);
    }
  };

  const handleFileSelect = (e: ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files && files.length > 0) {
      handleFiles(files);
    }
  };

  // Optimized state update functions using useCallback
  const updateImageProgress = useCallback((imageId: string, progress: number) => {
    setImages(prev => {
      const index = prev.findIndex(img => img.id === imageId);
      if (index === -1) return prev;
      
      const updated = [...prev];
      updated[index] = { ...updated[index], progress };
      return updated;
    });
  }, []);

  const updateImageStatus = useCallback((imageId: string, status: ImageFile['status'], error?: string) => {
    setImages(prev => {
      const index = prev.findIndex(img => img.id === imageId);
      if (index === -1) return prev;
      
      const updated = [...prev];
      updated[index] = { ...updated[index], status, error };
      return updated;
    });
  }, []);

  const removeImage = useCallback((id: string) => {
    setImages(prev => {
      const updated = prev.filter(img => img.id !== id);
      // Clean up preview URLs
      const removed = prev.find(img => img.id === id);
      if (removed) {
        URL.revokeObjectURL(removed.preview);
      }
      return updated;
    });
  }, []);

  const uploadImages = async () => {
    if (images.length === 0) {
      onError?.('Please select at least one image');
      return;
    }

    setIsUploading(true);

    try {
      // Request upload URLs
      const uploadResponse = await RecipeAPI.requestUpload(images.length);
      const { recipe_id, upload_urls } = uploadResponse;

      // Upload each image with better error handling
      const uploadPromises = images.map(async (image, index) => {
        const uploadUrl = upload_urls[index];
        
        // Update status to uploading using optimized function
        updateImageStatus(image.id, 'uploading');

        try {
          await RecipeAPI.uploadImage(
            uploadUrl, 
            image.file, 
            (progress) => {
              // Use optimized progress update function
              updateImageProgress(image.id, progress);
            }
          );

          // Mark as completed using optimized function
          updateImageStatus(image.id, 'completed');
          updateImageProgress(image.id, 100);
          
          return { success: true, imageId: image.id };
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Upload failed';
          // Mark as error using optimized function
          updateImageStatus(image.id, 'error', errorMessage);
          
          return { success: false, imageId: image.id, error: errorMessage };
        }
      });

      // Handle partial success - don't fail entire batch if some succeed
      const results = await Promise.allSettled(uploadPromises);
      const successes = results
        .filter(r => r.status === 'fulfilled' && r.value.success)
        .map(r => (r as PromiseFulfilledResult<{success: true; imageId: string}>).value);
      const failures = results
        .filter(r => r.status === 'fulfilled' && !r.value.success)
        .map(r => (r as PromiseFulfilledResult<{success: false; imageId: string; error: string}>).value);
      
      // Handle results appropriately
      if (failures.length === results.length) {
        // All failed
        throw new Error('All uploads failed');
      } else if (failures.length > 0) {
        // Partial success - show warning but continue
        console.warn(`${failures.length} of ${results.length} uploads failed`);
        onError?.(`${failures.length} of ${results.length} images failed to upload`);
      }
      
      // All uploads completed successfully
      onUploadComplete?.(recipe_id);
      
      // Clean up
      images.forEach(img => URL.revokeObjectURL(img.preview));
      setImages([]);
      
    } catch (error) {
      onError?.(error instanceof Error ? error.message : 'Upload failed');
    } finally {
      setIsUploading(false);
    }
  };

  const openFileDialog = () => {
    fileInputRef.current?.click();
  };

  const getStatusColor = (status: ImageFile['status']) => {
    switch (status) {
      case 'pending': return 'bg-gray-200';
      case 'uploading': return 'bg-blue-500';
      case 'completed': return 'bg-green-500';
      case 'error': return 'bg-red-500';
      default: return 'bg-gray-200';
    }
  };

  const getStatusText = (status: ImageFile['status']) => {
    switch (status) {
      case 'pending': return 'Ready';
      case 'uploading': return 'Uploading...';
      case 'completed': return 'Complete';
      case 'error': return 'Error';
      default: return '';
    }
  };

  return (
    <div className="w-full max-w-2xl mx-auto">
      {/* Drop Zone */}
      <div
        className={`
          relative border-2 border-dashed rounded-lg p-8 text-center transition-colors
          ${isDragOver 
            ? 'border-blue-500 bg-blue-50' 
            : 'border-gray-300 hover:border-gray-400'
          }
          ${isUploading ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
        `}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={openFileDialog}
      >
        <div className="space-y-4">
          <div className="mx-auto w-16 h-16 text-gray-400">
            <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} 
                    d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
            </svg>
          </div>
          
          <div>
            <p className="text-lg font-medium text-gray-900">
              Drop your recipe images here
            </p>
            <p className="text-sm text-gray-500 mt-1">
              or click to select files
            </p>
            <p className="text-xs text-gray-400 mt-2">
              PNG, JPG up to 10MB each • Maximum {maxImages} images
            </p>
          </div>
        </div>

        <input
          ref={fileInputRef}
          type="file"
          multiple
          accept="image/*"
          onChange={handleFileSelect}
          className="hidden"
        />
      </div>

      {/* Image Previews */}
      {images.length > 0 && (
        <div className="mt-6 space-y-4">
          <h3 className="text-lg font-medium text-gray-900">
            Selected Images ({images.length}/{maxImages})
          </h3>
          
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {images.map((image) => (
              <div key={image.id} className="relative border rounded-lg overflow-hidden">
                {/* Image Preview */}
                <div className="aspect-video bg-gray-100">
                  <img
                    src={image.preview}
                    alt="Recipe preview"
                    className="w-full h-full object-cover"
                  />
                </div>
                
                {/* Status Overlay */}
                <div className="absolute top-2 right-2">
                  {!isUploading && image.status === 'pending' && (
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        removeImage(image.id);
                      }}
                      className="bg-red-500 text-white rounded-full p-1 hover:bg-red-600 transition-colors"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  )}
                </div>
                
                {/* Progress Bar and Status */}
                <div className="p-3">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium text-gray-900 truncate flex-1 mr-2">
                      {image.file.name}
                    </span>
                    <span className={`text-xs px-2 py-1 rounded text-white ${getStatusColor(image.status)}`}>
                      {getStatusText(image.status)}
                    </span>
                  </div>
                  
                  {image.status === 'uploading' && (
                    <div className="w-full bg-gray-200 rounded-full h-2">
                      <div 
                        className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                        style={{ width: `${image.progress}%` }}
                      />
                    </div>
                  )}
                  
                  {image.status === 'error' && image.error && (
                    <p className="text-red-600 text-xs mt-1">{image.error}</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Upload Button */}
      {images.length > 0 && (
        <div className="mt-6">
          <button
            onClick={uploadImages}
            disabled={isUploading || images.some(img => img.status === 'uploading')}
            className="w-full bg-blue-600 text-white py-3 px-4 rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isUploading ? 'Uploading Images...' : `Upload ${images.length} Image${images.length > 1 ? 's' : ''}`}
          </button>
        </div>
      )}
    </div>
  );
}