'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import ImageUpload from '@/components/ImageUpload';

export default function AddRecipePage() {
  const router = useRouter();
  const [error, setError] = useState<string>('');
  const [success, setSuccess] = useState<string>('');
  const [isProcessing, setIsProcessing] = useState(false);

  const handleUploadComplete = (recipeId: number) => {
    setError('');
    setSuccess(`Recipe uploaded successfully! Your recipe is now being processed.`);
    setIsProcessing(true);
    
    // Redirect to the recipes list after a short delay
    setTimeout(() => {
      router.push('/recipes');
    }, 2000);
  };

  const handleUploadError = (errorMessage: string) => {
    setError(errorMessage);
    setSuccess('');
  };

  const clearMessages = () => {
    setError('');
    setSuccess('');
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-4xl mx-auto py-8 px-4 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="mb-8">
          <button
            onClick={() => router.push('/recipes')}
            className="flex items-center text-blue-600 hover:text-blue-800 mb-4 transition-colors"
          >
            <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
            Back to Recipes
          </button>
          
          <h1 className="text-3xl font-bold text-gray-900 mb-2">
            Add New Recipe
          </h1>
          <p className="text-gray-600 max-w-2xl">
            Upload photos of your recipe and our AI will extract the ingredients, instructions, and other details for you to review and edit.
          </p>
        </div>

        {/* Instructions */}
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-6 mb-8">
          <h2 className="text-lg font-semibold text-blue-900 mb-3">
            How it works:
          </h2>
          <ol className="list-decimal list-inside space-y-2 text-blue-800">
            <li>Upload clear photos of your recipe (recipe cards, cookbook pages, handwritten notes)</li>
            <li>Our AI will read the text and extract the recipe details</li>
            <li>Review and edit the extracted information</li>
            <li>Publish your organized recipe</li>
          </ol>
        </div>

        {/* Success Message */}
        {success && (
          <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-6">
            <div className="flex items-start">
              <svg className="w-5 h-5 text-green-400 mt-0.5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              <div>
                <h3 className="text-green-800 font-medium">Upload Successful!</h3>
                <p className="text-green-700 mt-1">{success}</p>
                {isProcessing && (
                  <p className="text-green-600 text-sm mt-2">
                    Redirecting to recipes list...
                  </p>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Error Message */}
        {error && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
            <div className="flex items-start">
              <svg className="w-5 h-5 text-red-400 mt-0.5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div>
                <h3 className="text-red-800 font-medium">Upload Error</h3>
                <p className="text-red-700 mt-1">{error}</p>
                <button
                  onClick={clearMessages}
                  className="text-red-600 hover:text-red-800 text-sm mt-2 underline"
                >
                  Dismiss
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Upload Component */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <ImageUpload
            onUploadComplete={handleUploadComplete}
            onError={handleUploadError}
            maxImages={5}
          />
        </div>

        {/* Tips */}
        <div className="mt-8 bg-gray-50 rounded-lg p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">
            Tips for better results:
          </h3>
          <div className="grid md:grid-cols-2 gap-4 text-sm text-gray-600">
            <div>
              <h4 className="font-medium text-gray-800 mb-2">Photo Quality</h4>
              <ul className="space-y-1">
                <li>• Ensure good lighting and clear text</li>
                <li>• Avoid shadows over the text</li>
                <li>• Take photos straight-on, not at an angle</li>
              </ul>
            </div>
            <div>
              <h4 className="font-medium text-gray-800 mb-2">What to Include</h4>
              <ul className="space-y-1">
                <li>• Recipe title and ingredients list</li>
                <li>• Step-by-step instructions</li>
                <li>• Serving size and cooking tips</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}