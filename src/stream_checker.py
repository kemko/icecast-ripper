import asyncio
from pprint import pprint
from aiohttp import ClientSession, ClientTimeout
from recorder import Recorder
from logger import log_event

class StreamChecker:
    def __init__(self, stream_url, check_interval, timeout_connect, output_directory, timeout_read=30):
        self.stream_url = stream_url
        self.check_interval = check_interval
        self.timeout_connect = timeout_connect
        self.timeout_read = timeout_read
        self.output_directory = output_directory
        self.recorder = None
        self.is_stream_live = False

    async def check_stream(self, session):
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
        except Exception as e:
            print(self.stream_url)
            log_event("check_stream_error", {"error": str(e)})

    async def run(self):
        while True:
            async with ClientSession() as session:
                await self.check_stream(session)

                if self.is_stream_live and (self.recorder is None or not self.recorder.is_active()):
                    self.recorder = Recorder(self.stream_url, self.output_directory, self.timeout_read)
                    await self.recorder.start_recording()

            await asyncio.sleep(self.check_interval)
