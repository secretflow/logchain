#!/usr/bin/env python3
"""
简单的日志接收服务，用于测试 Benthos 适配器
接收 POST /v1/logs 请求，将日志写入文件
"""

import os
import json
import datetime
from flask import Flask, request, jsonify
from werkzeug.exceptions import Unauthorized

app = Flask(__name__)

LOG_FILE = os.getenv('LOG_OUTPUT_FILE', 'ingested_logs.jsonl')
HOST = os.getenv('INGESTION_HOST', '0.0.0.0')
PORT = int(os.getenv('INGESTION_PORT', '8093'))


def validate_api_key():
    provided_key = request.headers.get('X-Api-Key')
    if provided_key is None:
        raise Unauthorized('Invalid API Key')
    return True


def write_log_to_file(data):
    timestamp = datetime.datetime.now().isoformat()

    log_dir = os.path.dirname(os.path.abspath(LOG_FILE))
    if log_dir and not os.path.exists(log_dir):
        os.makedirs(log_dir, exist_ok=True)

    with open(LOG_FILE, 'a', encoding='utf-8') as f:
        if isinstance(data, list):
            for item in data:
                log_entry = {
                    'received_at': timestamp,
                    'data': item
                }
                f.write(json.dumps(log_entry, ensure_ascii=False) + '\n')
        else:
            log_entry = {
                'received_at': timestamp,
                'data': data
            }
            f.write(json.dumps(log_entry, ensure_ascii=False) + '\n')
    
    print(f"[{timestamp}] Wrote {len(data) if isinstance(data, list) else 1} log entry/entries to {LOG_FILE}")


@app.route('/v1/logs', methods=['POST'])
def ingest_logs():
    """接收日志的端点"""
    try:
        validate_api_key()

        if not request.is_json:
            return jsonify({'error': 'Content-Type must be application/json'}), 400
        
        data = request.get_json()
        
        if data is None:
            return jsonify({'error': 'Invalid JSON'}), 400

        write_log_to_file(data)

        return jsonify({
            'status': 'success',
            'message': 'Logs ingested successfully'
        }), 200
        
    except Unauthorized as e:
        return jsonify({'error': str(e)}), 401
    except Exception as e:
        print(f"Error processing request: {e}")
        return jsonify({'error': 'Internal server error'}), 500


if __name__ == '__main__':
    print(f"""
    ========================================
    Log Ingestion Test Server
    ========================================
    Listening on: {HOST}:{PORT}
    Log file: {LOG_FILE}
    ========================================
    """)
    
    app.run(host=HOST, port=PORT, debug=True)
