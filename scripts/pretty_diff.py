#!/usr/bin/python
""" pretty_diff.py

    Generate pretty diff output of kube configs.

    Used as custom differ for "kubectl diff" calls. Written in Python instead of go since
    Python's difflib is really nice and I haven't been able to find a fully equivalent library
    in go.
"""

import difflib
import os
import os.path
import sys


class bcolors:
    OKBLUE = '\033[94m'
    OKGREEN = '\033[92m'
    FAIL = '\033[91m'
    ENDC = '\033[0m'


def main():
    if len(sys.argv) != 3:
        print('USAGE: pretty_diff.py [old path] [new path]')
        sys.exit(1)

    old_root = sys.argv[1]
    new_root = sys.argv[2]

    old_files = walk_path(old_root)
    new_files = walk_path(new_root)

    # Generate mapping between the two sets of files
    old_names = set()
    for old_file in old_files:
        base_name = os.path.relpath(old_file, old_root)
        old_names.add(base_name)

    new_names = set()
    for new_file in new_files:
        base_name = os.path.relpath(new_file, new_root)
        new_names.add(base_name)

    diff_pairs = []

    # Get use_colors from the environment instead of the command-line so it's easier
    # to use with kubectl.
    use_colors_env = os.environ.get('USE_COLORS', 'false')
    if use_colors_env.lower() == 'true':
        use_colors = True
    else:
        use_colors = False

    for name in old_names | new_names:
        old_path = os.path.join(old_root, name)
        new_path = os.path.join(new_root, name)

        if name in new_names and name in old_names:
            diff_pairs.append(
                (
                    name,
                    old_path,
                    new_path,
                ),
            )
        elif name in new_names:
            diff_pairs.append(
                (
                    name,
                    None,
                    new_path,
                ),
            )
        else:
            diff_pairs.append(
                (
                    name,
                    old_path,
                    None,
                ),
            )

    diff_pairs.sort(key=lambda r: r[0])

    for diff_pair in diff_pairs:
        diff_files(*diff_pair, use_colors=use_colors)


def diff_files(name, old_path, new_path, use_colors=True):
    old_lines = []
    new_lines = []

    if old_path is not None:
        with open(old_path, 'r') as old_file:
            for line in old_file:
                old_lines.append(line)

    if new_path is not None:
        with open(new_path, 'r') as new_file:
            for line in new_file:
                new_lines.append(line)

    diff = difflib.unified_diff(
        old_lines,
        new_lines,
        fromfile=os.path.join('api-server', name),
        tofile=os.path.join('local-configs', name),
    )
    for line in diff:
        line = line.rstrip()
        if line.startswith('+'):
            if use_colors:
                print(bcolors.OKGREEN + line + bcolors.ENDC)
            else:
                print(line)
        elif line.startswith('-'):
            if use_colors:
                print(bcolors.FAIL + line + bcolors.ENDC)
            else:
                print(line)
        elif line.startswith('^'):
            if use_colors:
                print(bcolors.OKBLUE + line + bcolors.ENDC)
            else:
                print(line)
        elif len(line) > 0:
            print(line)


def walk_path(root_path):
    if not os.path.exists(root_path):
        raise Exception('Path %s does not exist', root_path)

    if os.path.isfile(root_path):
        return [root_path]

    file_paths = []

    for dir_name, subdirs, files in os.walk(root_path):
        for file_name in files:
            file_paths.append(os.path.join(dir_name, file_name))

    return file_paths


if __name__ == "__main__":
    main()
