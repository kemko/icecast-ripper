"""Utility functions for the application"""
import hashlib
import string
from cache import get_cached_hash, cache_hash

def sanitize_filename(filename):
    """
    Sanitize the filename by removing or replacing invalid characters.
    """
    valid_chars = f"-_.() {string.ascii_letters}{string.digits}"
    cleaned_filename = "".join(c for c in filename if c in valid_chars)
    cleaned_filename = cleaned_filename.replace(' ', '_')  # Replace spaces with underscores
    return cleaned_filename

def generate_file_hash(file_path):
    """
    Generate a hash for file contents to uniquely identify files.
    """

    file_hash = get_cached_hash(file_path)

    if file_hash:
        return file_hash

    hasher = hashlib.sha256()
    with open(file_path, 'rb') as f:
        while chunk := f.read(8192):
            hasher.update(chunk)

    file_hash = hasher.hexdigest()
    cache_hash(file_path, file_hash)

    return file_hash

def file_hash_to_id(file_hash, length=32):
    """
    Convert file hash to a shorter file ID, considering only the first length characters.
    """
    return file_hash[:length]
