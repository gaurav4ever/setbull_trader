import numpy as np

def convert_numpy_types(obj):
    if isinstance(obj, dict):
        return {k: convert_numpy_types(v) for k, v in obj.items()}
    elif isinstance(obj, list):
        return [convert_numpy_types(item) for item in obj]
    elif isinstance(obj, np.generic):
        return obj.item()
    else:
        return obj
    
def round_string(string_value, precision=2):
    if isinstance(string_value, float):
        return round(string_value, precision)
    else:
        return round(float(string_value), precision)