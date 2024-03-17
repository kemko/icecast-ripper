"""cache"""

import sqlite3

def db_init(db_path='/tmp/recorded_file_hashes.db'):
    """
    Connect to the database and return the connection object.
    """

    conn = sqlite3.connect(db_path)
    conn.execute("CREATE TABLE IF NOT EXISTS file_hashes (file_path TEXT PRIMARY KEY, file_hash TEXT)")

    return conn

def get_cached_hash(file_path):
    """
    Cache the file hash in the database.
    """
    conn = db_init()

    # check if file_path exists then return it hash else insert
    cursor = conn.execute("SELECT file_hash FROM file_hashes WHERE file_path = ? LIMIT 1", (file_path,))
    row = cursor.fetchone()
    if row:
        conn.close()
        return row[0]

    return False

def cache_hash(file_path, file_hash):
    """
    Cache the file hash in the database.
    """

    try:
        conn = db_init()
        conn.execute("INSERT INTO file_hashes (file_path, file_hash) VALUES (?, ?)", (file_path, file_hash))
        conn.commit()
        conn.close()
        return True
    except sqlite3.IntegrityError as e:
        conn.close()
        print(e)
        return False
