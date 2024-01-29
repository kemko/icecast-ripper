"""Generates an RSS feed from the files in the output directory"""
import os
from datetime import datetime
from yattag import Doc
from utils import generate_file_hash, file_hash_to_id

def generate_rss_feed(files, output_directory, server_host):
    """Generates an RSS feed from the files in the output directory"""
    doc, tag, text = Doc().tagtext()

    doc.asis('<?xml version="1.0" encoding="UTF-8"?>')
    with tag('rss', version='2.0'):
        with tag('channel'):
            with tag('title'):
                text('Icecast Stream Recordings')
            with tag('description'):
                text('The latest recordings from the Icecast server.')
            with tag('link'):
                text(server_host)
            with tag('itunes:block'):
                text('yes')

            for file_name in files:
                file_path = os.path.join(output_directory, file_name)
                file_hash = generate_file_hash(file_path)
                file_id = file_hash_to_id(file_hash)

                with tag('item'):
                    with tag('enclosure', url=f'{server_host}/files/{file_name}', length=os.path.getsize(file_path), type='audio/mpeg'):
                        pass
                    with tag('title'):
                        text(file_name)
                    with tag('guid', isPermaLink='false'):
                        text(file_id)
                    with tag('pubDate'):
                        pub_date = datetime.utcfromtimestamp(os.path.getctime(file_path)).strftime('%a, %d %b %Y %H:%M:%S UTC')
                        text(pub_date)

    return doc.getvalue()
