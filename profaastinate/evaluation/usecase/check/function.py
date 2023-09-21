import os
import json
import subprocess
import time
import datetime
import sys
import base64
import requests
import uuid
from minio import Minio
from pdfminer.pdfparser import PDFParser
from pdfminer.pdfdocument import PDFDocument

def check(context, event):

    start_ts = time.time() * 1000
    # TODO change 'host.docker.internal' to 'localhost' on linux
    context.logger.debug("check function start")
    nuclioURL = "http://host.docker.internal:8070/api/function_invocations"
    minioURL = "host.docker.internal:9000"
    minioBucket = "profaastinate"

    # set missing headers (e.g., in case function was called synchronously)
    expected_headers = {
        "Callid": uuid.uuid4().hex,
        "X-Check-Filename": "fusionize.pdf",
        "X-Virus-Filename": "fusionize.pdf",
        "X-OCR-Filename": "fusionize.pdf",
        "X-Email-Filename": "fusionizeOCR.pdf",
        "Forceocr": True,
    }
    for header in expected_headers:
        if event.headers.get(header) is None:
            context.logger.debug(f"set missing header {header} to {expected_headers[header]}")
            event.headers[header] = expected_headers[header]

    # get filename to read from request header
    filename = "test.pdf" if event.headers.get("X-Check-Filename") is None else event.headers["X-Check-Filename"]
    deadline = "420000"
    context.logger.debug(f"filename={filename}")

    # get file
    minioClient = Minio(minioURL, access_key="minioadmin", secret_key="minioadmin", secure=False)
    minioClient.fget_object(minioBucket, filename, f"/tmp/{filename}")
    context.logger.debug("stored PDF from minio locally")

    # parse the PDF to access metadata
    parser = PDFParser(open(f"/tmp/{filename}", "rb"))
    document = PDFDocument(parser)
    context.logger.debug("parsed PDF")

    # call next function => virus check
    callid = event.headers["Callid"]
    response = requests.get(
        nuclioURL,
        headers={
            "x-nuclio-function-name": "virus",
            "x-nuclio-function-namespace": "nuclio",
            "x-nuclio-async": "true",
            "x-nuclio-async-deadline": deadline,
            "x-virus-filename": filename,
            "callid": callid
        }
    )
    context.logger.debug(response)
    context.logger.debug("check function end")

    # "profaastinate-request-timestamp", strconv.FormatInt(call.timestamp.UnixMilli(), 10))
    #		req.Header.Set("profaastinate-request-deadline"

    end_ts = time.time() * 1000
    eval_info = {
        "function": "check",
        "start": start_ts,
        "end": end_ts,
        "callid": callid
    }
    if event.headers.get("Profaastinate-Request-Timestamp"):
        eval_info["request_timestamp"] = event.headers["Profaastinate-Request-Timestamp"]
    if event.headers.get("Profaastinate-Request-Deadline"):
        eval_info["request_deadline"] = event.headers["Profaastinate-Request-Deadline"]
    if event.headers.get("Profaastinate-Mode"):
        eval_info["mode"] = event.headers["Profaastinate-Mode"]
    else:
        eval_info["mode"] = "sync"

    context.logger.warn(f"PFSTT{json.dumps(eval_info)}TTSFP")

    # return parsed metadata
    return context.Response(
        body=str(callid),
        headers={},
        content_type='text/plain',
        status_code=200
    )
