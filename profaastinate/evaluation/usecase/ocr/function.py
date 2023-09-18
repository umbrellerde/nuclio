import ocrmypdf
import requests
import uuid
import os
import json
import time

from pdfminer.pdfparser import PDFParser
from pdfminer.pdfdocument import PDFDocument
from minio import Minio


def ocr(context, event):

    start_ts = time.time() * 1000
    # TODO change 'host.docker.internal' to 'localhost' on linux
    context.logger.debug("ocr function start")
    nuclioURL = "http://host.docker.internal:8070/api/function_invocations"
    minioURL = "host.docker.internal:9000"
    minioBucket = "profaastinate"

    responseMsg = ""
    forceOCR = False


    # set missing headers (e.g., in case function was called synchronously)
    expected_headers = {
        "Callid": uuid.uuid4().hex,
        "X-Check-Filename": "fusionize.pdf",
        "X-Virus-Filename": "fusionize.pdf",
        "X-Ocr-Filename": "fusionize.pdf",
        "X-Email-Filename": "fusionizeOCR.pdf",
        "Forceocr": True,
    }
    for header in expected_headers:
        if event.headers.get(header) is None:
            context.logger.debug(f"set missing header {header} to {expected_headers[header]}")
            event.headers[header] = expected_headers[header]


    # get file (filename from header) from minio
    filename = "test.pdf" if event.headers.get("X-Ocr-Filename") is None else event.headers["X-Ocr-Filename"]
    deadline = "180000"
    context.logger.debug(f"filename={filename}")

    # TODO change 'host.docker.internal' to 'localhost' on linux
    pathIn, pathOut = f"/tmp/{filename}", f"/tmp/OCR_{filename}"
    minioClient = Minio(minioURL, access_key="minioadmin", secret_key="minioadmin", secure=False)
    minioClient.fget_object(minioBucket, filename, pathIn)
    context.logger.debug("stored PDF from minio locally")

    # do ocr
    try:
        # At some point, nuclio/python changes the capitalization of the header field; hence, 'Forceocr' instead of 'forceOCR'
        forceOCR = False
        if "Forceocr" in event.headers:
            forceOCR = bool(event.headers["Forceocr"])
        ocrmypdf.ocr(pathIn, pathOut, force_ocr=forceOCR)
        responseMsg = f"OCR successful! Stored output in '{pathOut}'"
    except ocrmypdf.PriorOcrFoundError as e:
        responseMsg = f"{str(e)}"

    # put OCR'd file into minio
    #minioClient.fput_object(minioBucket, f"OCR_{filename}", pathOut)

    # call next function
    callid = event.headers["Callid"]
    response = requests.get(
        nuclioURL,
        headers={
            "x-nuclio-function-name": "email",
            "x-nuclio-funcition-namespace": "nuclio",
            "x-nuclio-async": "true",
            "x-nuclio-async-deadline": deadline,
            "x-email-filename": f"OCR_{filename}",
            "callid": event.headers["Callid"]
        }
    )
    context.logger.debug(response)
    context.logger.debug("ocr function end")

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

    # return the encrypted body, and some hard-coded header
    return context.Response(body=responseMsg,
                            headers={'x-encrypt-algo': 'aes256'},
                            content_type='text/plain',
                            status_code=200)
