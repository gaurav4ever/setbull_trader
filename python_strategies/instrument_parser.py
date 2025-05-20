import json
import os

def filter_nse_index_objects(file_path):
    dir_name = os.path.dirname(file_path)
    base_name = os.path.basename(file_path)
    name_without_ext = os.path.splitext(base_name)[0]
    output_file = os.path.join(dir_name, f"{name_without_ext}_filtered.json")

    filtered_data = []

    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()

    decoder = json.JSONDecoder()
    idx = 0
    total = 0

    while idx < len(content):
        try:
            obj, next_idx = decoder.raw_decode(content, idx)
            idx = next_idx
            total += 1

            # Handle list of objects
            if isinstance(obj, list):
                for item in obj:
                    if isinstance(item, dict) and item.get("instrument_key", "").startswith("NSE_INDEX|"):
                        filtered_data.append(item)

            # Handle single dict object
            elif isinstance(obj, dict):
                if obj.get("instrument_key", "").startswith("NSE_INDEX|"):
                    filtered_data.append(obj)

        except json.JSONDecodeError as e:
            print(f"JSON decoding failed at position {idx}: {e}")
            break

    with open(output_file, 'w', encoding='utf-8') as out_f:
        json.dump(filtered_data, out_f, indent=2)

    print(f"Parsed {total} top-level JSON values, filtered {len(filtered_data)} NSE_INDEX objects to: {output_file}")


# # Example usage
# if __name__ == "__main__":
#     file_path = "/Users/gaurav/setbull_projects/setbull_trader/instruments.json"  # Change as needed
#     filter_nse_index_objects(file_path)


# Example usage
file_path = "/Users/gaurav/setbull_projects/setbull_trader/NSE.json"  # Replace with your actual file path
filter_nse_index_objects(file_path)
