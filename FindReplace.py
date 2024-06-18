import json
import argparse
import os

def replace_text_in_json(input_file, replace_dict):
    # Read the input JSON file
    with open(input_file, 'r') as file:
        data = json.load(file)

    # Function to recursively replace values in a nested dictionary or list
    def replace_values(obj, replace_dict):
        if isinstance(obj, dict):
            for key, value in obj.items():
                if isinstance(value, (dict, list)):
                    replace_values(value, replace_dict)
                elif isinstance(value, str):
                    for old_text, new_text in replace_dict.items():
                        obj[key] = obj[key].replace(old_text, new_text)
        elif isinstance(obj, list):
            for i in range(len(obj)):
                if isinstance(obj[i], (dict, list)):
                    replace_values(obj[i], replace_dict)
                elif isinstance(obj[i], str):
                    for old_text, new_text in replace_dict.items():
                        obj[i] = obj[i].replace(old_text, new_text)

    # Replace substrings based on replace_dict
    replace_values(data, replace_dict)

    return data

def save_json_to_file(data, output_file):
    # Write the modified content to the output JSON file
    with open(output_file, 'w') as file:
        json.dump(data, file, indent=4)

    print(f'Replacement done. Check {output_file} for the modified JSON.')

def main(input_dir, replace_dict):
    if not os.path.isdir(input_dir):
        print(f"Error: {input_dir} is not a valid directory.")
        return

    json_files = [f for f in os.listdir(input_dir) if f.endswith('.json') and 'Default' not in f]

    if not json_files:
        print(f"No JSON files found in the directory: {input_dir}")
        return

    for json_file in json_files:
        input_file = os.path.join(input_dir, json_file)
        modified_data = replace_text_in_json(input_file, replace_dict)
        
        # Write back to the same file
        save_json_to_file(modified_data, input_file)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Replace text in all JSON files in a directory based on a replacement dictionary.')
    parser.add_argument('--input_dir', type=str, required=True, help='Path to the input directory containing JSON files.')
    parser.add_argument('--replacements', type=str, required=True, help='Comma-separated key=value pairs for text replacement. Example: old1=new1,old2=new2')

    args = parser.parse_args()

    # Parse replacements argument into a dictionary
    replacements = dict(pair.split('=') for pair in args.replacements.split(','))

    main(args.input_dir, replacements)
