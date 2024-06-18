import json
import argparse
import os
import datetime


# Usage:
# python replace_json_text.py --input_file "path/to/input.json" --output_dir "path/to/output/directory" --replacements "old1=new1,old2=new2"


def replace_text_in_json(input_file, output_dir, replace_dict):
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

    # Create a dynamic output file name based on the input file name and current timestamp
    input_filename = os.path.basename(input_file)
    timestamp = datetime.datetime.now().strftime("%Y%m%d%H%M%S")
    output_filename = f"{os.path.splitext(input_filename)[0]}_{timestamp}.json"
    output_file = os.path.join(output_dir, output_filename)

    # Write the modified content to the output JSON file
    with open(output_file, 'w') as file:
        json.dump(data, file, indent=4)

    print(f'Replacement done. Check {output_file} for the modified JSON.')

# Example usage:
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Replace text in a JSON file based on a replacement dictionary.')
    parser.add_argument('--input_file', type=str, required=True, help='Path to the input JSON file.')
    parser.add_argument('--output_dir', type=str, required=True, help='Directory to save the output JSON file.')
    parser.add_argument('--replacements', type=str, required=True, help='Comma-separated key=value pairs for text replacement. Example: old1=new1,old2=new2')

    args = parser.parse_args()

    # Parse replacements argument into a dictionary
    replacements = dict(pair.split('=') for pair in args.replacements.split(','))

    replace_text_in_json(args.input_file, args.output_dir, replacements)
