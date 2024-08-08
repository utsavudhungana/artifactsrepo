#!/bin/bash

# Variables
api_version=2020-12-01
source_endpoint=https://$source_workspace_name.dev.azuresynapse.net
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

# List artifact type and store in an array, removing spaces
mapfile -t var_array < <(az synapse $aztype list --workspace-name $source_workspace_name --query "[].name" -o tsv | sed 's/ //g')

# Collect all search and replace pairs into an array
search_replace_pairs=("${@:2}")

# Loop over the array to get each artifact type and save as .json file
for var in "${var_array[@]}"; do
    curl -X GET -H "Authorization: Bearer $(az account get-access-token --resource=https://dev.azuresynapse.net/ --query accessToken --output tsv)" $source_endpoint/$1s/$var?api-version=$api_version > "${var}.json"

    # Replace substrings in the JSON file if search and replace pairs are provided
    if [[ ${#search_replace_pairs[@]} -gt 0 ]]; then
        replace_subtext_in_json "${var}.json" search_replace_pairs[@]
    fi
done

# Loop over the array and put each artifact type to target workspace
for var in "${var_array[@]}"; do
    if [[ $1 == "linkedservice" && $var == "$source_workspace_name-WorkspaceDefaultSqlServer" ]] || [[ $1 == "linkedservice" && $var == "$source_workspace_name-WorkspaceDefaultStorage" ]]; then
        continue
    else
        curl -X PUT -d @$var.json -H "Authorization: Bearer $(az account get-access-token --resource=https://dev.azuresynapse.net/ --query accessToken --output tsv)" $target_endpoint/$1s/$var?api-version=$api_version
    fi
done

# Clean up
for var in "${var_array[@]}"; do
    rm $var.json
done