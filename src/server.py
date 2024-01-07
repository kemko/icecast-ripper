"""Server module for the application"""
import os
import mimetypes
from pathlib import Path
from aiohttp import web
import logger
from rss_generator import generate_rss_feed

routes = web.RouteTableDef()

def log_request(request, level="INFO"):
    """Log an HTTP request"""
    logger.log_event("http_request", {
        "method": request.method,
        "path": request.path,
        "remote": request.remote,
        "query_string": request.query_string,
        "headers": dict(request.headers)
    }, level)

@routes.get('/health')
async def helth_check(request):
    """Health check endpoint"""
    log_request(request)
    return web.Response(text="OK")

@routes.get('/rss')
async def rss_feed(request):
    """RSS feed endpoint"""
    log_request(request)
    output_directory = request.app['config'].output_directory
    files = [f for f in os.listdir(output_directory) if f.endswith('.mp3')]
    rss_xml = generate_rss_feed(files, output_directory, request.app['config'].server_host)
    return web.Response(text=rss_xml, content_type='application/rss+xml')

@routes.get('/files/{file_name}')
async def serve_file(request):
    """File serving endpoint"""
    file_name = request.match_info['file_name']
    log_request(request)

    output_directory = request.app['config'].output_directory
    file_path = os.path.join(output_directory, file_name)

    if not Path(output_directory).joinpath(file_name).resolve().relative_to(Path(output_directory).resolve()):
        logger.log_event("file_access_denied", {"file_name": file_name}, level="WARNING")
        return web.Response(status=403, text='Access denied')

    if not os.path.exists(file_path):
        logger.log_event("file_not_found", {"file_name": file_name}, level="WARNING")
        return web.Response(status=404, text='File not found')

    file = os.path.basename(file_path)
    content_type, _ = mimetypes.guess_type(file)

    headers = {
        'Content-Type': content_type or 'application/octet-stream',
    }
    return web.FileResponse(file_path, headers=headers)

async def start_server(config):
    """Start the web server"""
    app = web.Application()
    app['config'] = config
    app.add_routes(routes)
    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, '0.0.0.0', config.server_port)
    logger.log_event('server_starting', {'port': config.server_port}, level="DEBUG")
    await site.start()
