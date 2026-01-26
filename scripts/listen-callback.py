#!/usr/bin/env python3
"""
Simple HTTP server to receive and display Digital Twin callbacks
Listens on port 8082 for maintenance completion notifications
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
from datetime import datetime

class CallbackHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path == '/api/v1/maintenance-completed':
            content_length = int(self.headers['Content-Length'])
            body = self.rfile.read(content_length)
            
            print("\n" + "="*60)
            print(f"CALLBACK RECEIVED at {datetime.now().strftime('%H:%M:%S')}")
            print("="*60)
            
            try:
                data = json.loads(body)
                print("\nPayload:")
                print(json.dumps(data, indent=2))
                
                print("\nSummary:")
                print(f"   Order ID: {data.get('maintenanceOrder', 'N/A')}")
                print(f"   Status: {data.get('status', 'N/A')}")
                print(f"   Equipment: {data.get('equipment', 'N/A')}")
                print(f"   Plant: {data.get('plant', 'N/A')}")
                
                if 'operations' in data and data['operations']:
                    print(f"\n   Operations ({len(data['operations'])}):")
                    for op in data['operations']:
                        print(f"      - {op.get('text', 'N/A')}")
                        if 'components' in op and op['components']:
                            print(f"        Components ({len(op['components'])}):")
                            for comp in op['components']:
                                print(f"          • {comp.get('material', 'N/A')}: {comp.get('description', 'N/A')}")
                
                if 'objectList' in data and data['objectList']:
                    print(f"\n   Object List ({len(data['objectList'])}):")
                    for obj in data['objectList']:
                        print(f"      - Equipment: {obj.get('equipment', 'N/A')}")
                        print(f"        Serial: {obj.get('serialNumber', 'N/A')}")
                
            except json.JSONDecodeError:
                print("\nERROR: Invalid JSON payload")
                print(body.decode('utf-8'))
            
            # Send 200 OK response
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(b'{"status": "received"}')
            
        else:
            # 404 for other paths
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        # Suppress default HTTP logging
        pass

def run_server(port=8082):
    server_address = ('', port)
    httpd = HTTPServer(server_address, CallbackHandler)
    
    print("="*60)
    print("Digital Twin Callback Listener")
    print("="*60)
    print(f"\nListening on port {port}")
    print(f"Endpoint: POST http://localhost:{port}/api/v1/maintenance-completed")
    print("\nWaiting for callbacks from SAP Adaptor...")
    print("Press Ctrl+C to stop\n")
    
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\n\nShutting down...")
        httpd.shutdown()

if __name__ == '__main__':
    run_server()
