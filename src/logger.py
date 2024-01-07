"""This module contains the logger functions for the application"""
import json
import sys
from datetime import datetime
from config import load_configuration

# Log levels
LOG_LEVELS = {
    "DEBUG": 10,
    "INFO": 20,
    "WARNING": 30,
    "ERROR": 40,
    "FATAL": 50
}

def log_event(event, details, level="INFO"):
    """Log an event to stdout in JSON format"""
    config = load_configuration()
    config_level_name = config.log_level.upper()

    if config_level_name not in LOG_LEVELS:
        raise ValueError(f"Invalid log level {config_level_name} in configuration")

    config_level_number = LOG_LEVELS[config_level_name]
    event_log_level = LOG_LEVELS.get(level.upper(), 20)  # Defaults to INFO if level is invalid

    if event_log_level >= config_level_number:
        log_entry = {
            "timestamp": datetime.utcnow().isoformat(),
            "event": event,
            "level": level.upper(),
            "details": details
        }
        json_log_entry = json.dumps(log_entry)
        print(json_log_entry, file=sys.stdout)
        sys.stdout.flush()  # Immediately flush the log entry
