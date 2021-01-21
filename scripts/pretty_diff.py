#!/usr/bin/python
""" pretty_diff.py

    Generate pretty diff output of kube configs.

    Used as custom differ for "kubectl diff" calls. Written in Python instead of go since
    Python's difflib is really nice and I haven't been able to find a fully equivalent library
    in go.

    TODO: Try porting to go and using https://github.com/pmezard/go-difflib?
"""

import difflib
import os
import os.path
import re
import sys


MAX_LINE_LEN = 256

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

    strip_managed_fields_env = os.environ.get('STRIP_MANAGED_FIELDS', 'false')
    if strip_managed_fields_env.lower() == 'true':
        strip_managed_fields = True
    else:
        strip_managed_fields = False

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
        diff_files(
            *diff_pair,
            use_colors=use_colors,
            strip_managed_fields=strip_managed_fields)


def diff_files(
        name, old_path, new_path,
        use_colors=True,
        strip_managed_fields=False):
    old_lines = []
    new_lines = []

    if old_path is not None:
        old_lines = get_lines(
            old_path, strip_managed_fields=strip_managed_fields)

    if new_path is not None:
        new_lines = get_lines(
            new_path, strip_managed_fields=strip_managed_fields)

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


def get_lines(path, strip_managed_fields=False):
    inside_managed_fields = False

    lines = []

    with open(path, 'r') as input_file:
        for line in input_file:
            keep = True

            if strip_managed_fields:
                # Managed fields begin with a 'managedFields:' and end when we hit the next
                # top-level metadata field (usually name).
                if line.startswith('  managedFields:'):
                    inside_managed_fields = True
                    keep = False
                elif inside_managed_fields:
                    if not (line.startswith('  -') or line.startswith('   ')):
                        inside_managed_fields = False
                    else:
                        keep = False

            if keep:
                # Trim very long lines
                if len(line) > MAX_LINE_LEN:
                    lines.append(
                        line[0:MAX_LINE_LEN] +
                        '... (%d chars omitted)' % (len(line)-MAX_LINE_LEN),
                    )
                else:
                    lines.append(line)

    return lines


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
