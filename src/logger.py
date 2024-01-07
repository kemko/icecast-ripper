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
    print(json_log_entry, file=sys.stdout)
    sys.stdout.flush()  # Immediately flush the log entry
