import os
import argparse

def generate_tree(startpath, exclude_dirs=None, max_depth=None):
    """
    Generate a directory tree, with options to exclude specific directories 
    and limit depth.
    
    :param startpath: Root directory to start the tree from
    :param exclude_dirs: List of directory names to exclude
    :param max_depth: Maximum depth of the tree to display
    """
    # Normalize exclude_dirs to handle both relative and absolute paths
    if exclude_dirs is None:
        exclude_dirs = []
    exclude_dirs = [os.path.normpath(d) for d in exclude_dirs]
    
    def _tree(directory, prefix='', depth=0):
        # Check if we've exceeded max depth
        if max_depth is not None and depth > max_depth:
            return
        
        # Get contents of the directory
        try:
            contents = os.listdir(directory)
        except PermissionError:
            print(f"{prefix}⚠️ [Permission Denied]")
            return
        except FileNotFoundError:
            print(f"{prefix}⚠️ [Directory Not Found]")
            return
        
        # Sort contents for consistent output
        contents.sort()
        
        # Iterate through directory contents
        for i, item in enumerate(contents):
            # Construct full path
            full_path = os.path.normpath(os.path.join(directory, item))
            
            # Check if this item should be excluded
            if any(excluded in full_path for excluded in exclude_dirs):
                continue
            
            # Determine tree connector based on whether it's the last item
            is_last = (i == len(contents) - 1)
            connector = '└── ' if is_last else '├── '
            
            # Print the current item
            if os.path.isdir(full_path):
                print(f"{prefix}{connector}{item}/")
                
                # Recursively print subdirectories
                extension = '    ' if is_last else '│   '
                _tree(full_path, prefix=prefix+extension, depth=depth+1)
            else:
                print(f"{prefix}{connector}{item}")

    # Start the tree from the given path
    print(os.path.abspath(startpath) + '/')
    _tree(startpath)

def main():
    # Set up argument parsing
    parser = argparse.ArgumentParser(description='Generate a directory tree with exclusion options.')
    parser.add_argument('directory', nargs='?', default='.', 
                        help='Directory to generate tree for (default: current directory)')
    parser.add_argument('-x', '--exclude', nargs='+', 
                        help='Directories to exclude (can use multiple)')
    parser.add_argument('-d', '--depth', type=int, 
                        help='Maximum depth of the tree')
    
    # Parse arguments
    args = parser.parse_args()
    
    # Generate the tree
    generate_tree(args.directory, args.exclude, args.depth)

if __name__ == '__main__':
    main()

# Example usage:
# python directory_tree.py              # Current directory
# python directory_tree.py /path/to/dir # Specific directory
# python directory_tree.py -x .git node_modules # Exclude .git and node_modules
# python directory_tree.py -d 2         # Limit to 2 levels deep
# python directory_tree.py -x .git -d 3  # Combine exclusions and depth limit