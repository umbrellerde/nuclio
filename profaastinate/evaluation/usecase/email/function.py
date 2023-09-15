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

    # read header fields
    filename = "testOCR.pdf" if event.headers.get("X-Email-Filename") is None else event.headers["X-Email-Filename"]
    deadline = 180000
    context.logger.debug(f"filename={filename}, calltime={calltime}, deadline={deadline}")

    # read & print pdf from minio
    minioClient = Minio(minioURL, access_key="minioadmin", secret_key="minioadmin", secure=False)
    minioClient.fget_object(minioBucket, filename, f"/tmp/{filename}")
    pdfContent = getPDFString(f"/tmp/{filename}")
    print(pdfContent)

    context.logger.debug("email function end")

    end_ts = time.time() * 1000
    eval_info = {
        "function": "email",
        "start": start_ts,
        "end": end_ts,
        "request_timestamp": event.headers["Profaastinate-Request-Timestamp"],
        "request_deadline": event.headers["Profaastinate-Request-Deadline"],
        "mode": event.headers["Profaastinate-Mode"]
    }
    context.logger.warn(f"PFSTT{json.dumps(eval_info)}TTSFP")

    # answer http request
    return context.Response(
        body=pdfContent,
        status_code=200,
        content_type='text/plain'
    )
