#!/usr/bin/python
""" cluster_summary.py

    Print out a summary of a cluster. Used for "kubeapply status" calls.
    Written in Python for simplicity, but could be rewritten in go and incorporated into
    the kube client directly.
"""

import argparse
import logging
import os
import subprocess

import tabulate

logging.basicConfig(
    format='%(asctime)s %(levelname)s %(message)s',
    level=logging.INFO,
)


class bcolors:
    HEADER = '\033[95m'
    OKBLUE = '\033[94m'
    OKGREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'


def main():
    args = get_args()

    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)

    print_heading('PODS', no_color=args.no_color)
    pods_text = kubectl_get_text('pods', args.namespace, args.kubeconfig)
    pretty_table(pods_text)

    print_heading('JOBS', no_color=args.no_color)
    jobs_text = kubectl_get_text('jobs', args.namespace, args.kubeconfig)
    pretty_table(jobs_text)

    print_heading('DEPLOYMENTS', no_color=args.no_color)
    deployments_text = kubectl_get_text(
        'deployments', args.namespace, args.kubeconfig)
    pretty_table(deployments_text)

    print_heading('STATEFULSETS', no_color=args.no_color)
    statefulsets_text = kubectl_get_text(
        'statefulsets', args.namespace, args.kubeconfig)
    pretty_table(statefulsets_text)

    print_heading('DAEMONSETS', no_color=args.no_color)
    daemonsets_text = kubectl_get_text(
        'daemonsets', args.namespace, args.kubeconfig)
    pretty_table(daemonsets_text)

    print_heading('NODES', no_color=args.no_color)
    nodes_text = kubectl_get_text('nodes', args.namespace, args.kubeconfig)
    pretty_table(nodes_text)


def kubectl_get_text(resource_type, namespace, kubeconfig):
    cmd = [
        'kubectl',
        'get',
        resource_type,
        '--kubeconfig',
        kubeconfig,
    ]

    if resource_type != 'nodes':
        if namespace != '':
            cmd += [
                '-n',
                namespace,
            ]
        else:
            cmd.append('--all-namespaces')

    logging.debug('Running command: %s', str(cmd))

    return subprocess.check_output(cmd).decode('utf-8')


def kubectl_get_json(resource_type, namespace, kubeconfig):
    cmd = [
        'kubectl',
        'get',
        resource_type,
        '--output',
        'json'
        '--kubeconfig',
        kubeconfig,
    ]

    if resource_type != 'nodes':
        if namespace != '':
            cmd += [
                '-n',
                namespace,
            ]
        else:
            cmd.append('--all-namespaces')

    logging.debug('Running command: %s', str(cmd))

    result = subprocess.check_output(cmd)
    return json.loads(result.decode('utf-8'))


def pretty_table(text):
    lines = text.split('\n')

    split_lines = []

    for l, line in enumerate(lines):
        if line == '':
            continue

        if l == 0:
            line = line.replace('NODE SELECTOR', 'NODE_SELECTOR')

        components = line.split(' ')
        split_lines.append([c for c in components if c != ''])

    if len(split_lines) == 0:
        print('None found\n')
        return

    print(tabulate.tabulate(
        split_lines[1:],
        headers=split_lines[0],
        showindex=True,
    ))
    print('')


def print_heading(heading, no_color=False):
    if no_color:
        print('>>> %s' % heading)
    else:
        print(bcolors.OKBLUE + bcolors.BOLD + '    %s' %
              heading + bcolors.ENDC)


def get_args():
    parser = argparse.ArgumentParser(
        description='Get summary of the state of a cluster')
    parser.add_argument(
        '--debug',
        default=False,
        action='store_true',
        help='Turn on debug logging',
    )
    parser.add_argument(
        '--no-color',
        default=False,
        action='store_true',
        help='Turn off colors',
    )
    parser.add_argument(
        '-n',
        '--namespace',
        type=str,
        default='',
        help='Namespace to restrict to',
    )
    parser.add_argument(
        '--all-namespaces',
        default=False,
        action='store_true',
        help='Get data for all namespaces',
    )
    parser.add_argument(
        '--kubeconfig',
        type=str,
        default=os.environ.get('KUBECONFIG', ''),
        help='Kubeconfig',
    )

    return parser.parse_args()


if __name__ == "__main__":
    main()
