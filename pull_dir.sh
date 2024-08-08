#!/bin/bash

#Pre-Requisite: Azure Cli and Docker Cli installed
#Usage: Replace variables values and use command "./pull_dir.sh" in bash. This script will pull docker image from registry, run it as a container and copy its content to the directory specified.
#Author: Utsav Dhungana

# Variables
ACR_NAME="synapseacr"
IMAGE_NAME="myapp"
IMAGE_TAG="v1"
LOCAL_OUTPUT_DIR="./Extracted/"

# Login to ACR
az acr login --name $ACR_NAME

# Pull the Docker image from ACR
docker pull $ACR_NAME.azurecr.io/$IMAGE_NAME:$IMAGE_TAG

# Create a container from the pulled image
CONTAINER_ID=$(docker create $ACR_NAME.azurecr.io/$IMAGE_NAME:$IMAGE_TAG)

# Create local output directory if it doesn't exist
mkdir -p $LOCAL_OUTPUT_DIR

# Copy the entire directory from the container to the local output directory
docker cp $CONTAINER_ID:/data $LOCAL_OUTPUT_DIR

# Clean up the container
docker rm $CONTAINER_ID

# Output the location of the extracted files
echo "Directory extracted to: $LOCAL_OUTPUT_DIR"
