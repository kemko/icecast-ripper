"""Recorder class for recording a stream to a file"""
import os
from datetime import datetime, timedelta
import aiohttp
from logger import log_event
from utils import sanitize_filename

class Recorder: # pylint: disable=too-many-instance-attributes
    """Recorder class for recording a stream to a file"""
    def __init__(self, stream_url, output_directory, timeout_connect=10, timeout_read=30):
        self.stream_url = stream_url
        self.output_directory = output_directory
        self.timeout_read = timeout_read
        self.timeout_connect = timeout_connect
        self.file_name = None
        self.file_path = None
        self.start_time = None
        self.last_data_time = None
        self.is_recording = False

    async def start_recording(self):
        """Start recording the stream to a file"""

        if not os.path.exists(self.output_directory):
            try:
                os.makedirs(self.output_directory)
            except Exception as e: # pylint: disable=broad-except
                log_event("output_directory_error", {"error": str(e)}, level="ERROR")

        self.start_time = datetime.utcnow()
        domain = self.stream_url.split("//")[-1].split("/")[0]
        sanitized_domain = sanitize_filename(domain)
        date_str = self.start_time.strftime("%Y%m%d_%H%M%S")
        self.file_name = f"{sanitized_domain}_{date_str}.mp3.tmp"
        self.file_path = os.path.join(self.output_directory, self.file_name)
        try:
            timeout = aiohttp.ClientTimeout(total=None, connect=self.timeout_connect, sock_read=self.timeout_read)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                async with session.get(self.stream_url) as response:
                    if response.status == 200:
                        self.is_recording = True
                        log_event("recording_started", {"file_name": self.file_name, "stream_url": self.stream_url})
                        async for data, _ in response.content.iter_chunks():
                            if not data:
                                break
                            self.last_data_time = datetime.utcnow()
                            with open(self.file_path, 'ab') as f:
                                f.write(data)
                            # Check if timeout exceeded between data chunks
                            if datetime.utcnow() - self.last_data_time > timedelta(seconds=self.timeout_read):
                                log_event("timeout_exceeded", {
                                    "stream_url": self.stream_url,
                                    "elapsed_seconds": (datetime.utcnow() - self.last_data_time).total_seconds()
                                }, level="WARNING")
                                break

                        log_event("recording_finished", {"file_name": self.file_name, "stream_url": self.stream_url})
                    else:
                        log_event("stream_unavailable", {"http_status": response.status})
        except Exception as e: # pylint: disable=broad-except
            log_event('recording_error', {"error": str(e)}, level="ERROR")
        finally:
            self.is_recording = False
            self.end_recording()

    def end_recording(self):
        """Rename the temporary file to a finished file"""
        if os.path.exists(self.file_path):
            finished_file = self.file_path.replace('.tmp', '')
            os.rename(self.file_path, finished_file)
            log_event("recording_saved", {
                "file_name": finished_file,
                "duration": (datetime.utcnow() - self.start_time).total_seconds() if self.start_time else 0
            })

    def is_active(self):
        """Check if the recorder is currently recording a stream"""
        return self.is_recording
