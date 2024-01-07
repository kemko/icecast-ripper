"""Server module for the application"""
import os
import mimetypes
from pprint import pprint
from pathlib import Path
from aiohttp import web
from rss_generator import generate_rss_feed
from logger import log_event

routes = web.RouteTableDef()

@routes.get('/health')
async def helth_check(request):
    """Health check endpoint"""
    log_event("health_check_requested", {"method": "GET", "path": request.path}, level="INFO")
    return web.Response(text="OK")

@routes.get('/rss')
async def rss_feed(request):
    """RSS feed endpoint"""
    log_event("rss_feed_requested", {"method": "GET", "path": request.path}, level="INFO")
    output_directory = request.app['config'].output_directory
    files = [f for f in os.listdir(output_directory) if f.endswith('.mp3')]
    rss_xml = generate_rss_feed(files, output_directory, request.app['config'].server_host)
    return web.Response(text=rss_xml, content_type='application/rss+xml')

@routes.get('/files/{file_name}')
async def serve_file(request):
    """File serving endpoint"""
    file_name = request.match_info['file_name']
    log_event("file_serve_requested", {"method": "GET", "path": request.path, "file_name": file_name}, level="INFO")

    output_directory = request.app['config'].output_directory
    file_path = os.path.join(output_directory, file_name)
    pprint(file_path)

    if not Path(output_directory).joinpath(file_name).resolve().relative_to(Path(output_directory).resolve()):
        log_event("file_access_denied", {"file_name": file_name}, level="WARNING")
        return web.Response(status=403, text='Access denied')

    if not os.path.exists(file_path):
        log_event("file_not_found", {"file_name": file_name}, level="WARNING")
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
    log_event('server_starting', {'port': config.server_port}, level="INFO")
    await site.start()
    log_event('server_started', {'port': config.server_port}, level="INFO")
