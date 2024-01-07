"""Main entry point for the Icecast stream checker and recorder"""
import asyncio
from server import start_server
from stream_checker import StreamChecker
from config import load_configuration
from logger import log_event

def main():
    """Main entry point for the Icecast stream checker and recorder"""
    # Load configuration from command line arguments and environment variables
    config = load_configuration()

    log_event("service_start", {"config": config.__dict__}, level="DEBUG")

    # Create the StreamChecker instance
    checker = StreamChecker(
        stream_url=config.stream_url,
        check_interval=config.check_interval,
        timeout_connect=config.timeout_connect,
        timeout_read=config.timeout_read,
        output_directory=config.output_directory
    )

    # Start the Icecast stream checking and recording loop
    checker_task = asyncio.ensure_future(checker.run())

    # Start the health check and file serving server
    server_task = asyncio.ensure_future(start_server(config))

    # Run both tasks in the event loop
    loop = asyncio.get_event_loop()
    try:
        loop.run_until_complete(asyncio.gather(checker_task, server_task))
    except KeyboardInterrupt:
        pass
    finally:
        checker_task.cancel()
        server_task.cancel()
        loop.close()

if __name__ == "__main__":
    main()
