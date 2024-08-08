#!/bin/bash

#Usage: 
#command: ./fetch_artifacts.sh sqlScript 
#Note: (replace sqlScript with above with artifact type you need to fetch). Make sure to setup env var source_workspace_name

# Variables
api_version=2020-12-01
source_endpoint=https://$source_workspace_name.dev.azuresynapse.net

# Determine artifact type
if [[ $1 == "sqlScript" ]]; then
    aztype="sql-script"
elif [[ $1 == "linkedservice" ]]; then
    aztype="linked-service"
elif [[ $1 == "sparkJobDefinition" ]]; then
    aztype="spark-job-definition"
else
    aztype=$1
fi

# Create directory structure for artifacts
mkdir -p artifacts/$aztype

# List artifact type and store in an array, removing spaces
mapfile -t var_array < <(az synapse $aztype list --workspace-name $source_workspace_name --query "[].name" -o tsv | sed 's/ //g')

# Loop over the array to get each artifact type and save as .json file
for var in "${var_array[@]}"; do
    curl -X GET -H "Authorization: Bearer $(az account get-access-token --resource=https://dev.azuresynapse.net/ --query accessToken --output tsv)" $source_endpoint/$1s/$var?api-version=$api_version > "artifacts/$aztype/${var}.json"
done

# Optional: Commit and push changes to GitLab
git add artifacts/$aztype/*
git commit -m "Save $aztype artifacts"
git push origin main