#!/bin/bash

#Usage:
#command: ./deploy_artifacts.sh sqlScript "search_text_1" "replace_text_1" "search_text_2" "replace_text_2"

# Variables
api_version=2020-12-01
target_endpoint=https://$workspace_name.dev.azuresynapse.net

# Function to replace substrings in the JSON file
replace_subtext_in_json() {
    local json_file=$1
    local search_replace_pairs=("${!2}")

    # Loop over search_replace_pairs array in steps of 2
    for ((i=0; i<${#search_replace_pairs[@]}; i+=2)); do
        search_text=${search_replace_pairs[i]}
        replace_text=${search_replace_pairs[i+1]}
        sed -i "s/${search_text}/${replace_text}/g" "$json_file"
    done
}

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

# Directory containing the JSON files
artifact_dir="artifacts/$aztype"

# Check if directory exists
if [[ ! -d "$artifact_dir" ]]; then
    echo "Directory $artifact_dir does not exist. Exiting."
    exit 1
fi

# Collect all search and replace pairs into an array
search_replace_pairs=("${@:2}")

# Loop over the JSON files and deploy to the target workspace
for json_file in "$artifact_dir"/*.json; do
    var=$(basename "$json_file" .json)

    # Replace substrings in the JSON file if search and replace pairs are provided
    if [[ ${#search_replace_pairs[@]} -gt 0 ]]; then
        replace_subtext_in_json "$json_file" search_replace_pairs[@]
    fi

    # Skip certain linked services (if applicable)
    if [[ $1 == "linkedservice" && $var == "$source_workspace_name-WorkspaceDefaultSqlServer" ]] || [[ $1 == "linkedservice" && $var == "$source_workspace_name-WorkspaceDefaultStorage" ]]; then
        continue
    else
        curl -X PUT -d @$json_file -H "Authorization: Bearer $(az account get-access-token --resource=https://dev.azuresynapse.net/ --query accessToken --output tsv)" $target_endpoint/$1s/$var?api-version=$api_version
    fi
done