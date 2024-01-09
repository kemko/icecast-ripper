"""Module for loading configuration values from command line arguments, environment variables and defaults"""

import argparse
import os
from dotenv import load_dotenv

# Load .env file if available
load_dotenv()

# Default configuration values
DEFAULTS = {
    'server_host': 'https://example.org',
    'server_port': 8080,
    'stream_url': 'http://example.com/stream',
    'output_directory': './records',
    'check_interval': 60,
    'timeout_connect': 10,
    'timeout_read': 30,
    'log_level': 'info'
}

def parse_arguments():
    """Parse command line arguments"""
    parser = argparse.ArgumentParser(description='Icecast Ripper Service')
    parser.add_argument('--server-host', help='Server host name with protocol')
    parser.add_argument('--server-port', type=int, help='Server port number')
    parser.add_argument('--stream-url', help='URL of the Icecast stream to monitor and record')
    parser.add_argument('--file-url-base', help='Base URL used for constructing file links in the RSS feed')
    parser.add_argument('--output-directory', help='Directory to save the recordings')
    parser.add_argument('--check-interval', type=int, help='Interval to check the stream in seconds')
    parser.add_argument('--timeout-connect', type=int, help='Timeout for connecting to the stream in seconds')
    parser.add_argument('--timeout-read', type=int, help='Read timeout in seconds')
    parser.add_argument('--log-level', help='Log level')
    return vars(parser.parse_args())

def load_configuration():
    """Get values from command line arguments, environment variables and defaults"""
    cmd_args = parse_arguments()

    # Configuration is established using a priority: CommandLine > EnvironmentVars > Defaults
    config = {
        'server_host': cmd_args['server_host'] or os.getenv('SERVER_HOST') or DEFAULTS['server_host'],
        'server_port': cmd_args['server_port'] or os.getenv('SERVER_PORT') or DEFAULTS['server_port'],
        'stream_url': cmd_args['stream_url'] or os.getenv('STREAM_URL') or DEFAULTS['stream_url'],
        'output_directory': cmd_args['output_directory'] or os.getenv('OUTPUT_DIRECTORY') or DEFAULTS['output_directory'],
        'check_interval': cmd_args['check_interval'] or os.getenv('CHECK_INTERVAL') or DEFAULTS['check_interval'],
        'timeout_connect': cmd_args['timeout_connect'] or os.getenv('TIMEOUT_CONNECT') or DEFAULTS['timeout_connect'],
        'timeout_read': cmd_args['timeout_read'] or os.getenv('TIMEOUT_READ') or DEFAULTS['timeout_read'],
        'log_level': cmd_args['log_level'] or os.getenv('LOG_LEVEL') or DEFAULTS['log_level']
    }

    # Converting string paths to absolute paths
    config['output_directory'] = os.path.abspath(config['output_directory'])

    return argparse.Namespace(**config)
