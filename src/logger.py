import json
import sys
from datetime import datetime

# Log levels
DEBUG = "DEBUG"
INFO = "INFO"
WARNING = "WARNING"
ERROR = "ERROR"
FATAL = "FATAL"

def log_event(event, details, level=INFO):
    log_entry = {
        "timestamp": datetime.utcnow().isoformat(),
        "event": event,
        "level": level,
        "details": details
    }
    json_log_entry = json.dumps(log_entry)
    print(json_log_entry, file=sys.stdout)  # Write to stdout
    sys.stdout.flush()  # Immediately flush the log entry

# Specific log functions per level for convenience
def log_debug(event, details):
    log_event(event, details, level=DEBUG)

def log_info(event, details):
    log_event(event, details, level=INFO)

def log_warning(event, details):
    log_event(event, details, level=WARNING)

def log_error(event, details):
    log_event(event, details, level=ERROR)

def log_fatal(event, details):
    log_event(event, details, level=FATAL)
