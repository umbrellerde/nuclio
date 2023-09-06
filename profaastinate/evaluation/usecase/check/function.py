# this code runs as a nuclio function. Thus, context and event are filled by the nuclio runtime.
# Import everything we need to upload a file to minio, and to read the metadata of a pdf file
import os
import json
import subprocess
import time
import datetime
import sys
import base64
import minio
from pdfminer.pdfparser import PDFParser
from pdfminer.pdfdocument import PDFDocument


# context and event are passed by nuclio. context.logger is used to log to the nuclio console, event.body contains a base64 encoded string of the pdf file
def entrypoint(context, event):
    # load the pdf into a file
    with open("/tmp/file.pdf", "wb") as file:
        file.write(base64.b64decode(event.body))
    
    # get the metadata of the pdf file
    parser = PDFParser(open("/tmp/file.pdf", "rb"))
    document = PDFDocument(parser)

    print(document.info)