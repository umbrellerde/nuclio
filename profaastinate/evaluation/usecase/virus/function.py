# this code runs as a nuclio function. Thus, context and event are filled by the nuclio runtime.
# Import everything we need to upload a file to minio, and to read the metadata of a pdf file
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


# context and event are passed by nuclio. context.logger is used to log to the nuclio console, event.body contains a base64 encoded string of the pdf file
def virus(context, event):

    start_ts = time.time() * 1000

    # TODO change 'host.docker.internal' to 'localhost' on linux
    context.logger.debug("virus function start")
    context.logger.debug(event.headers)
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
    filename = "test.pdf" if event.headers.get("X-Virus-Filename") is None else event.headers["X-Virus-Filename"]
    deadline = "180000"
    context.logger.debug(f"filename={filename}")

    # download the file "file.pdf" from minio using default credentials
    client = Minio(minioURL, access_key="minioadmin", secret_key="minioadmin", secure=False)
    client.fget_object(minioBucket, filename, f"/tmp/{filename}")
    context.logger.debug("stored PDF from minio locally")

    # read file using pdfparser
    parser = PDFParser(open(f"/tmp/{filename}", "rb"))
    document = PDFDocument(parser)

    # perform "virus check" on file
    # 1. calculate sha256 hash of file
    sha256 = subprocess.run(["sha256sum", f"/tmp/{filename}"], stdout=subprocess.PIPE).stdout.decode('utf-8').split(" ")[0]
    print("(use case) sha256: " + sha256)

    # 2. check if hash is in the list of known hashes
    #    if it is, return "virus found"
    #    if it is not, return "no virus found"
    if sha256 in ["d41d8cd98f00b204e9800998ecf8427e", "b026324c6904b2a9cb4b88d6d61c81d1",
                  "26ab0db90d72e28ad0ba1e22ee510510", "6d7fce9fee471194aa8b5b6e47267f03"]:
        print("(use case) virus found")
        return context.Response(body="virus found", headers={}, content_type='text/plain', status_code=200)

    # call the next function in the pipeline. Its called ocr, and is in the same project
    # the function is called with the file name in the body
    # TODO the function is called asynchronously, so we don't wait for a response
    #context.invoke("ocr", body="file.pdf") # TODO
    callid = event.headers["Callid"]
    response = requests.get(
        nuclioURL,
        headers={
            "x-nuclio-function-name": "ocr",
            "x-nuclio-funcition-namespace": "nuclio",
            "x-nuclio-async": "true",
            "x-nuclio-async-deadline": deadline,
            "x-ocr-filename": filename,
            "callid": event.headers["Callid"]
        }
    )
    context.logger.debug(response)
    context.logger.debug("virus function end")

    end_ts = time.time() * 1000
    eval_info = {
        "function": "virus",
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

    return context.Response(
        status_code=200,
        body="no virus found"
    )
