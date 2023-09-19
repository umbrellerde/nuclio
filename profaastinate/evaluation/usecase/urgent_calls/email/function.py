from minio import Minio
from io import StringIO
from pdfminer.converter import TextConverter
from pdfminer.layout import LAParams
from pdfminer.pdfdocument import PDFDocument
from pdfminer.pdfinterp import PDFResourceManager, PDFPageInterpreter
from pdfminer.pdfpage import PDFPage
from pdfminer.pdfparser import PDFParser
import os
import json
import time
import uuid

def getPDFString(fileIn):
    output_string = StringIO()
    with open(fileIn, 'rb') as in_file:
        parser = PDFParser(in_file)
        doc = PDFDocument(parser)
        rsrcmgr = PDFResourceManager()
        device = TextConverter(rsrcmgr, output_string, laparams=LAParams())
        interpreter = PDFPageInterpreter(rsrcmgr, device)
        for page in PDFPage.create_pages(doc):
            interpreter.process_page(page)
    return output_string.getvalue()

def email(context, event):

    start_ts = time.time() * 1000
    minioURL = "host.docker.internal:9000"
    minioBucket = "profaastinate"

    context.logger.debug("email function start")
    context.logger.debug(event.body)

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

    # read header fields
    filename = "testOCR.pdf" if event.headers.get("X-Email-Filename") is None else event.headers["X-Email-Filename"]
    context.logger.debug(f"filename={filename}")

    # read & print pdf from minio
    minioClient = Minio(minioURL, access_key="minioadmin", secret_key="minioadmin", secure=False)
    minioClient.fget_object(minioBucket, filename, f"/tmp/{filename}")
    pdfContent = getPDFString(f"/tmp/{filename}")
    print(pdfContent)

    context.logger.debug("email function end")

    end_ts = time.time() * 1000
    eval_info = {
        "function": "urgentemail",
        "start": start_ts,
        "end": end_ts,
        "callid": event.headers["Callid"]
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

    # answer http request
    return context.Response(
        body=pdfContent,
        status_code=200,
        content_type='text/plain'
    )
