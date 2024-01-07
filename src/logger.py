"""This module contains the logger functions for the application"""
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
    """Log an event to stdout in JSON format"""
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
    """Log a debug event"""
    log_event(event, details, level=DEBUG)

def log_info(event, details):
    """Log an info event"""
    log_event(event, details, level=INFO)

def log_warning(event, details):
    """Log a warning event"""
    log_event(event, details, level=WARNING)

def log_error(event, details):
    """Log an error event"""
    log_event(event, details, level=ERROR)

def log_fatal(event, details):
    """Log a fatal event"""
    log_event(event, details, level=FATAL)
