#!/bin/bash
# Google Cloud Storage Bucket Setup Script for Digital Recipes
# This script creates and configures a GCS bucket with proper security policies

set -e

# Configuration
PROJECT_ID=${GOOGLE_CLOUD_PROJECT:-"digital-recipes"}
BUCKET_NAME=${GCS_BUCKET_NAME:-"digital-recipes-images"}
LOCATION=${GCS_LOCATION:-"us-central1"}
SERVICE_ACCOUNT_NAME="digital-recipes-storage"

echo "🚀 Setting up Google Cloud Storage for Digital Recipes"
echo "Project ID: $PROJECT_ID"
echo "Bucket Name: $BUCKET_NAME"
echo "Location: $LOCATION"
echo ""

# Check if gcloud is installed and authenticated
if ! command -v gcloud &> /dev/null; then
    echo "❌ Error: gcloud CLI is not installed"
    echo "Please install Google Cloud SDK: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Check authentication
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q .; then
    echo "❌ Error: No active gcloud authentication found"
    echo "Please run: gcloud auth login"
    exit 1
fi

# Set project
echo "📋 Setting Google Cloud project..."
gcloud config set project $PROJECT_ID

# Enable required APIs
echo "🔧 Enabling required Google Cloud APIs..."
gcloud services enable storage-api.googleapis.com
gcloud services enable iam.googleapis.com
gcloud services enable cloudresourcemanager.googleapis.com

# Create storage bucket with security settings
echo "🪣 Creating storage bucket: $BUCKET_NAME"
if gcloud storage buckets describe gs://$BUCKET_NAME &> /dev/null; then
    echo "✅ Bucket already exists: gs://$BUCKET_NAME"
else
    gcloud storage buckets create gs://$BUCKET_NAME \
        --project=$PROJECT_ID \
        --location=$LOCATION \
        --default-storage-class=STANDARD \
        --uniform-bucket-level-access \
        --public-access-prevention
    echo "✅ Created bucket: gs://$BUCKET_NAME"
fi

# Configure bucket lifecycle management
echo "♻️ Setting up bucket lifecycle management..."
cat > /tmp/lifecycle.json << EOF
{
  "lifecycle": {
    "rule": [
      {
        "action": {"type": "Delete"},
        "condition": {
          "age": 365,
          "matchesPrefix": ["recipes/"]
        }
      },
      {
        "action": {"type": "SetStorageClass", "storageClass": "NEARLINE"},
        "condition": {
          "age": 30,
          "matchesPrefix": ["recipes/"]
        }
      }
    ]
  }
}
EOF

gcloud storage buckets update gs://$BUCKET_NAME \
    --lifecycle-file=/tmp/lifecycle.json
rm /tmp/lifecycle.json
echo "✅ Configured lifecycle management"

# Configure CORS for web uploads
echo "🌐 Setting up CORS policy..."
cat > /tmp/cors.json << EOF
[
  {
    "origin": ["http://localhost:3000", "https://digital-recipes.app"],
    "method": ["GET", "PUT", "OPTIONS"],
    "responseHeader": ["Content-Type", "x-goog-meta-*"],
    "maxAgeSeconds": 3600
  }
]
EOF

gcloud storage buckets update gs://$BUCKET_NAME \
    --cors-file=/tmp/cors.json
rm /tmp/cors.json
echo "✅ Configured CORS policy"

# Create service account for the application
echo "👤 Creating service account..."
if gcloud iam service-accounts describe $SERVICE_ACCOUNT_NAME@$PROJECT_ID.iam.gserviceaccount.com &> /dev/null; then
    echo "✅ Service account already exists: $SERVICE_ACCOUNT_NAME"
else
    gcloud iam service-accounts create $SERVICE_ACCOUNT_NAME \
        --description="Service account for Digital Recipes storage operations" \
        --display-name="Digital Recipes Storage"
    echo "✅ Created service account: $SERVICE_ACCOUNT_NAME"
fi

# Grant minimal required permissions
echo "🔐 Setting up IAM permissions..."
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT_NAME@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/storage.objectAdmin"

# Wait a moment for service account propagation
echo "⏳ Waiting for service account propagation..."
sleep 10

# Create and download service account key
KEY_FILE="$HOME/.config/gcloud/digital-recipes-storage-key.json"
mkdir -p "$(dirname "$KEY_FILE")"

if [ -f "$KEY_FILE" ]; then
    echo "✅ Service account key already exists: $KEY_FILE"
else
    gcloud iam service-accounts keys create "$KEY_FILE" \
        --iam-account=$SERVICE_ACCOUNT_NAME@$PROJECT_ID.iam.gserviceaccount.com
    chmod 600 "$KEY_FILE"
    echo "✅ Created service account key: $KEY_FILE"
fi

# Create bucket notification for processing queue (optional for Phase 3)
echo "📢 Setting up bucket notifications (for future AI processing)..."
# Note: This will be used in Phase 3 for triggering AI processing
# gcloud pubsub topics create recipe-upload-notifications
# gcloud storage buckets notifications create gs://$BUCKET_NAME \
#     --topic=recipe-upload-notifications \
#     --event-types=OBJECT_FINALIZE

echo ""
echo "🎉 Google Cloud Storage setup complete!"
echo ""
echo "Environment variables for your application:"
echo "export GOOGLE_CLOUD_PROJECT=\"$PROJECT_ID\""
echo "export GCS_BUCKET_NAME=\"$BUCKET_NAME\""
echo "export GOOGLE_APPLICATION_CREDENTIALS=\"$KEY_FILE\""
echo ""
echo "Add these to your .env file or export them in your shell:"
cat > .env.gcs << EOF
# Google Cloud Storage Configuration
GOOGLE_CLOUD_PROJECT=$PROJECT_ID
GCS_BUCKET_NAME=$BUCKET_NAME
GOOGLE_APPLICATION_CREDENTIALS=$KEY_FILE
EOF

echo "✅ Environment configuration saved to .env.gcs"
echo ""
echo "Next steps:"
echo "1. Source the environment: source .env.gcs"
echo "2. Restart your API service to use Google Cloud Storage"
echo "3. Test the upload endpoint with the configured bucket"