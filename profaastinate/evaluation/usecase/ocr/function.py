import ocrmypdf
import requests
from pdfminer.pdfparser import PDFParser
from pdfminer.pdfdocument import PDFDocument
from minio import Minio
import os
import json
import time

def ocr(context, event):

    start_ts = time.time() * 1000
    # TODO change 'host.docker.internal' to 'localhost' on linux
    context.logger.debug("ocr function start")
    nuclioURL = "http://host.docker.internal:8070/api/function_invocations"
    minioURL = "host.docker.internal:9000"
    minioBucket = "profaastinate"

    responseMsg = ""
    forceOCR = False

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
    response = requests.get(
        nuclioURL,
        headers={
            "x-nuclio-function-name": "email",
            "x-nuclio-funcition-namespace": "nuclio",
            "x-nuclio-async": "true",
            "x-nuclio-async-deadline": deadline,
            "x-email-filename": f"OCR_{filename}",
            "callid": event.header["Callid"]
        }
    )
    context.logger.debug(response)
    context.logger.debug("ocr function end")

    end_ts = time.time() * 1000
    eval_info = {
        "function": "ocr",
        "start": start_ts,
        "end": end_ts,
        "request_timestamp": event.headers["Profaastinate-Request-Timestamp"],
        "request_deadline": event.headers["Profaastinate-Request-Deadline"],
        "mode": event.headers["Profaastinate-Mode"],
        "callid": event.header["Callid"]
    }
    context.logger.warn(f"PFSTT{json.dumps(eval_info)}TTSFP")

    # return the encrypted body, and some hard-coded header
    return context.Response(body=responseMsg,
                            headers={'x-encrypt-algo': 'aes256'},
                            content_type='text/plain',
                            status_code=200)
