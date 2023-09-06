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
    # download the file "file.pdf" from minio using default credentials
    client = minio.Minio('minio:9000',access_key="minioadmin",secret_key="minioadmin",secure=False)
    client.fget_object("files", "file.pdf", "/tmp/file.pdf")

    # read file using pdfparser
    parser = PDFParser(open("/tmp/file.pdf", "rb"))
    document = PDFDocument(parser)

    # perform "virus check" on file
    # 1. calculate sha256 hash of file
    sha256 = subprocess.run(["sha256sum", "/tmp/file.pdf"], stdout=subprocess.PIPE).stdout.decode('utf-8').split(" ")[0]
    print("(use case) sha256: " + sha256)

    # 2. check if hash is in the list of known hashes
    #    if it is, return "virus found"
    #    if it is not, return "no virus found"
    if sha256 in ["d41d8cd98f00b204e9800998ecf8427e", "b026324c6904b2a9cb4b88d6d61c81d1", "26ab0db90d72e28ad0ba1e22ee510510", "6d7fce9fee471194aa8b5b6e47267f03"]:
        print("(use case) virus found")
        return context.Response(body="virus found", headers={}, content_type='text/plain', status_code=200)
    
    # call the next function in the pipeline. Its called ocr, and is in the same project
    # the function is called with the file name in the body
    # the function is called asynchronously, so we don't wait for a response
    context.invoke("ocr", body="file.pdf")