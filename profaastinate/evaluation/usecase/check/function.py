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

    # get filename to read from request header
    filename = "test.pdf" if event.headers.get("X-Check-Filename") is None else event.headers["X-Check-Filename"]
    deadline = "180000"
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
    callid = uuid.uuid4().hex
    response = requests.get(
        nuclioURL,
        headers={
            "x-nuclio-function-name": "virus",
            "x-nuclio-funcition-namespace": "nuclio",
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
        "request_timestamp": event.headers["Profaastinate-Request-Timestamp"],
        "request_deadline": event.headers["Profaastinate-Request-Deadline"],
        "mode": event.headers["Profaastinate-Mode"],
        "callid": callid
    }
    context.logger.warn(f"PFSTT{json.dumps(eval_info)}TTSFP")

    # return parsed metadata
    return context.Response(
        body=str(document.info),
        headers={},
        content_type='text/plain',
        status_code=200
    )
