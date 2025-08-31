# Google Cloud Storage Setup for Digital Recipes

This guide walks you through setting up Google Cloud Storage (GCS) for secure image uploads in the Digital Recipes application.

## Prerequisites

1. **Google Cloud Account**: You need access to Google Cloud Console
2. **Google Cloud SDK**: Install `gcloud` CLI tool
3. **Project Setup**: Have a Google Cloud project created

## Quick Setup (Automated)

For a complete automated setup, run the provided script:

```bash
# Make sure you're in the api-service directory
cd /home/filipe-carneiro/projects/digital-recipes/api-service

# Run the setup script
./scripts/setup-gcs-bucket.sh
```

The script will:
- ✅ Enable required Google Cloud APIs
- ✅ Create a secure storage bucket with lifecycle management
- ✅ Configure CORS for web uploads
- ✅ Create a service account with minimal permissions
- ✅ Generate and download service account credentials
- ✅ Output environment variables for your application

## Manual Setup (Step-by-Step)

### 1. Install Google Cloud SDK

```bash
# Install gcloud CLI (if not already installed)
# Visit: https://cloud.google.com/sdk/docs/install

# Authenticate with Google Cloud
gcloud auth login

# Set your project ID
export PROJECT_ID="digital-recipes"
gcloud config set project $PROJECT_ID
```

### 2. Enable Required APIs

```bash
gcloud services enable storage-api.googleapis.com
gcloud services enable iam.googleapis.com
```

### 3. Create Storage Bucket

```bash
# Set configuration
export BUCKET_NAME="digital-recipes-images"
export LOCATION="us-central1"

# Create bucket with security settings
gcloud storage buckets create gs://$BUCKET_NAME \
    --project=$PROJECT_ID \
    --location=$LOCATION \
    --storage-class=STANDARD \
    --uniform-bucket-level-access \
    --public-access-prevention
```

### 4. Configure Bucket Lifecycle

Create a lifecycle policy to manage costs:

```bash
cat > lifecycle.json << EOF
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

gcloud storage buckets update gs://$BUCKET_NAME --lifecycle-file=lifecycle.json
```

### 5. Setup CORS Policy

Configure CORS for frontend web uploads:

```bash
cat > cors.json << EOF
[
  {
    "origin": ["http://localhost:3000", "https://your-domain.com"],
    "method": ["GET", "PUT", "OPTIONS"],
    "responseHeader": ["Content-Type", "x-goog-meta-*"],
    "maxAgeSeconds": 3600
  }
]
EOF

gcloud storage buckets update gs://$BUCKET_NAME --cors-file=cors.json
```

### 6. Create Service Account

```bash
# Create service account
gcloud iam service-accounts create digital-recipes-storage \
    --description="Service account for Digital Recipes storage operations" \
    --display-name="Digital Recipes Storage"

# Grant storage permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:digital-recipes-storage@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/storage.objectAdmin"

# Create and download key
gcloud iam service-accounts keys create ~/.config/gcloud/digital-recipes-storage-key.json \
    --iam-account=digital-recipes-storage@$PROJECT_ID.iam.gserviceaccount.com
```

## Application Configuration

### Environment Variables

Add these environment variables to your application:

```bash
# Required GCS Configuration
export GOOGLE_CLOUD_PROJECT="your-project-id"
export GCS_BUCKET_NAME="digital-recipes-images"
export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.config/gcloud/digital-recipes-storage-key.json"
```

### Using .env File

Copy the example environment file and update it:

```bash
cp .env.example .env
# Edit .env with your actual values
```

Example .env configuration:
```env
# Google Cloud Storage Configuration
GOOGLE_CLOUD_PROJECT=your-project-id
GCS_BUCKET_NAME=digital-recipes-images
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
```

## Security Best Practices

### Bucket Security
- ✅ **Uniform bucket-level access**: Prevents object-level ACLs
- ✅ **Public access prevention**: Blocks public internet access
- ✅ **CORS restrictions**: Only allows specific origins
- ✅ **Lifecycle management**: Automatic cleanup and cost optimization

### Service Account Security
- ✅ **Minimal permissions**: Only `storage.objectAdmin` role
- ✅ **Scoped to specific bucket**: No project-wide access
- ✅ **Key rotation**: Regularly rotate service account keys
- ✅ **Secure storage**: Keep key file permissions at 600

### Network Security
- ✅ **Pre-signed URLs**: Limited-time upload access
- ✅ **Content validation**: File type and size restrictions
- ✅ **IP logging**: Track upload sources in metadata
- ✅ **Rate limiting**: Prevent abuse through API limits

## Testing Your Setup

### Test Bucket Access

```bash
# Test bucket exists and is accessible
gsutil ls gs://$BUCKET_NAME

# Test service account permissions
gcloud auth activate-service-account --key-file=$GOOGLE_APPLICATION_CREDENTIALS
gsutil ls gs://$BUCKET_NAME
```

### Test API Integration

```bash
# Start your API service
go run main.go

# Check health endpoint (should show storage status)
curl http://localhost:8080/health

# Test upload request endpoint (requires authentication)
curl -X POST http://localhost:8080/api/v1/recipes/upload-request \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-jwt-token" \
  -d '{"image_count": 1}'
```

## Monitoring and Maintenance

### View Bucket Usage

```bash
# Check bucket size and object count
gsutil du -sh gs://$BUCKET_NAME

# List recent uploads
gsutil ls -l gs://$BUCKET_NAME/recipes/
```

### View Logs

```bash
# View Cloud Storage logs
gcloud logging read "resource.type=gcs_bucket AND resource.labels.bucket_name=$BUCKET_NAME" \
  --limit=50 --format="table(timestamp,severity,jsonPayload.message)"
```

### Cost Optimization

- **Lifecycle rules**: Automatically delete old files
- **Storage classes**: Move infrequently accessed files to cheaper storage
- **Monitoring**: Set up billing alerts for unexpected usage

## Troubleshooting

### Common Issues

1. **"Access Denied" errors**
   - Check service account has `storage.objectAdmin` role
   - Verify `GOOGLE_APPLICATION_CREDENTIALS` path is correct
   - Ensure bucket exists and is in the correct project

2. **CORS errors in frontend**
   - Verify CORS policy includes your frontend domain
   - Check preflight OPTIONS requests are allowed
   - Ensure frontend sends correct headers

3. **Pre-signed URL generation fails**
   - Confirm service account has signing permissions
   - Check bucket uniform access is enabled
   - Verify expiration time is reasonable (< 7 days)

### Debug Commands

```bash
# Check service account authentication
gcloud auth list

# Test bucket access
gsutil acl get gs://$BUCKET_NAME

# Verify CORS configuration
gsutil cors get gs://$BUCKET_NAME

# Check lifecycle policy
gsutil lifecycle get gs://$BUCKET_NAME
```

## Production Considerations

### High Availability
- Use multi-regional storage class for critical data
- Implement retry logic for transient failures
- Consider multiple buckets in different regions

### Security
- Rotate service account keys regularly
- Use Google Cloud Secret Manager for sensitive values
- Enable audit logging for compliance
- Implement Content Security Policy (CSP)

### Performance
- Choose bucket location close to your users
- Use appropriate storage classes
- Implement client-side compression
- Cache metadata when possible

## Support

For issues specific to this setup:
1. Check the troubleshooting section above
2. Review Google Cloud Storage documentation
3. Check application logs for detailed error messages
4. Verify all environment variables are set correctly

For Google Cloud specific issues:
- [Google Cloud Storage Documentation](https://cloud.google.com/storage/docs)
- [Google Cloud Support](https://cloud.google.com/support)