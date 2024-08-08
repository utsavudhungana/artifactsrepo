#!/bin/bash

#Pre-Requisite: Azure Cli and Docker Cli installed
#Usage: Replace variables values and use command "./push_dir.sh" in bash. This script will push directory content to acr as docker image.
#Author: Utsav Dhungana

# Variables
RESOURCE_GROUP="rg-syna-learning"
ACR_NAME="synapseacr"
IMAGE_NAME="myapp"
IMAGE_TAG="v1"
DIRECTORY_TO_PUSH="./"

# Login to ACR
az acr login --name $ACR_NAME

# Create a temporary Dockerfile
cat <<EOF > Dockerfile
FROM busybox
COPY $DIRECTORY_TO_PUSH /data/
CMD ["ls", "-la", "/data"]
EOF

# Build the Docker image
docker build -t $ACR_NAME.azurecr.io/$IMAGE_NAME:$IMAGE_TAG .

# Push the Docker image to ACR
docker push $ACR_NAME.azurecr.io/$IMAGE_NAME:$IMAGE_TAG

# Clean up temporary Dockerfile
rm Dockerfile

echo "Docker image $ACR_NAME.azurecr.io/$IMAGE_NAME:$IMAGE_TAG has been pushed to ACR."
