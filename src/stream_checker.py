"""Checking the stream status and starting the ripper"""
import asyncio
from aiohttp import ClientSession, ClientTimeout
from ripper import Ripper
from logger import log_event

class StreamChecker:
    """Checking the stream status and starting the ripper"""
    def __init__(self, stream_url, check_interval, timeout_connect, output_directory, timeout_read=30): # pylint: disable=too-many-arguments
        self.stream_url = stream_url
        self.check_interval = check_interval
        self.timeout_connect = timeout_connect
        self.timeout_read = timeout_read
        self.output_directory = output_directory
        self.ripper = None
        self.is_stream_live = False

    async def check_stream(self, session):
        """Check if the stream is live and start the ripper if needed"""
        try:
            timeout = ClientTimeout(connect=self.timeout_connect)
            async with session.get(self.stream_url, timeout=timeout, allow_redirects=True) as response:
                if response.status == 200:
                    self.is_stream_live = True
                    log_event("stream_live", {"stream_url": self.stream_url})
                else:
                    self.is_stream_live = False
                    log_event("stream_offline", {"stream_url": self.stream_url})
        except asyncio.TimeoutError:
            log_event("check_stream_timeout", {"stream_url": self.stream_url})
        except Exception as e: # pylint: disable=broad-except
            print(self.stream_url)
            log_event("check_stream_error", {"error": str(e)})

    async def run(self):
        """Start the stream checking and recording loop"""
        while True:
            async with ClientSession() as session:
                await self.check_stream(session)

                if self.is_stream_live and (self.ripper is None or not self.ripper.is_active()):
                    self.ripper = Ripper(self.stream_url, self.output_directory, self.timeout_read)
                    await self.ripper.start_recording()

            await asyncio.sleep(int(self.check_interval))
