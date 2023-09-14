from minio import Minio
from io import StringIO
from pdfminer.converter import TextConverter
from pdfminer.layout import LAParams
from pdfminer.pdfdocument import PDFDocument
from pdfminer.pdfinterp import PDFResourceManager, PDFPageInterpreter
from pdfminer.pdfpage import PDFPage
from pdfminer.pdfparser import PDFParser


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

    minioURL = "host.docker.internal:9000"
    minioBucket = "profaastinate"
    context.logger.debug("email function started")

    # read header fields
    filename = "testOCR.pdf" if event.headers.get("X-Read-Filename") is None else event.headers["X-Read-Filename"]
    calltime = event.headers.get("Calltime") # TODO what happens if this header is missing?
    deadline = event.headers.get("Deadline") # TODO what happens if this header is missing?
    context.logger.debug(f"filename={filename}, calltime={calltime}, deadline={deadline}")

    # read & print pdf from minio
    minioClient = Minio(minioURL, access_key="minioadmin", secret_key="minioadmin", secure=False)
    minioClient.fget_object(minioBucket, filename, f"/tmp/{filename}")
    pdfContent = getPDFString(f"/tmp/{filename}")

    context.logger.info(f"PDF content:\n{pdfContent}")
    context.logger.debug("email function ended")

    # answer http request
    return context.Response(
        body=pdfContent,
        status_code=200,
        content_type='text/plain'
    )
