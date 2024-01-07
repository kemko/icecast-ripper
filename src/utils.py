import hashlib
import string

def sanitize_filename(filename):
    """
    Sanitize the filename by removing or replacing invalid characters.
    """
    valid_chars = "-_.() %s%s" % (string.ascii_letters, string.digits)
    cleaned_filename = "".join(c for c in filename if c in valid_chars)
    cleaned_filename = cleaned_filename.replace(' ', '_')  # Replace spaces with underscores
    return cleaned_filename

def generate_file_hash(file_path):
    """
    Generate a hash for file contents to uniquely identify files.
    """
    hasher = hashlib.sha256()
    with open(file_path, 'rb') as f:
        while chunk := f.read(8192):
            hasher.update(chunk)
    return hasher.hexdigest()

def file_hash_to_id(file_hash, length=32):
    """
    Convert file hash to a shorter file ID, considering only the first length characters.
    """
    return file_hash[:length]
