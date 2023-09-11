import os
import json
import subprocess
import time
import datetime
import sys
import base64
from minio import Minio
from pdfminer.pdfparser import PDFParser
from pdfminer.pdfdocument import PDFDocument

def check(context, event):

    context.logger.debug("starting function")

    # get filename to read from request header
    if 'X-Check-Filename' in event.headers:
        filename = event.headers['X-Check-Filename']
    else:
        context.logger.warn("no filename to retrieve from minio, using 'test.pdf'")
        filename = "test.pdf"

    # get file
    # TODO change 'host.docker.internal' to 'localhost' on linux (?)
    minioClient = Minio("host.docker.internal:9000", access_key="minioadmin", secret_key="minioadmin", secure=False)
    file = minioClient.fget_object("profaastinate", filename, "/tmp/file.pdf")

    # parse the PDF to access metadata
    parser = PDFParser(open("/tmp/file.pdf", "rb"))
    document = PDFDocument(parser)

    # return parsed metadata
    return context.Response(
        body=str(document.info), 
        headers={}, 
        content_type='text/plain', 
        status_code=200
        )
