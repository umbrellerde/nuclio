import ocrmypdf
from pdfminer.pdfparser import PDFParser
from pdfminer.pdfdocument import PDFDocument
from minio import Minio

def ocr(context, event):

    responseMsg = ""
    forceOCR = False

    # get file (filename from header) from minio
    if 'X-Ocr-Filename' in event.headers:
        filename = event.headers['X-Ocr-Filename']
    else:
        context.logger.warn("no filename to retrieve from minio, using 'test.pdf'")
        filename = "test.pdf"
    # TODO change 'host.docker.internal' to 'localhost' on linux
    pathIn, pathOut = "/tmp/input.pdf", "/tmp/output.pdf"
    minioClient = Minio("host.docker.internal:9000",access_key="minioadmin",secret_key="minioadmin",secure=False)
    file = minioClient.fget_object("profaastinate", filename, pathIn)

    context.logger.debug("got file")

    # do ocr
    try:
        # At some point, nuclio/python changes the capitalization of the header field; hence, 'Forceocr' instead of 'forceOCR' :(
        if "Forceocr" in event.headers:
            forceOCR = bool(event.headers["Forceocr"])
        ocrmypdf.ocr(pathIn, pathOut, force_ocr=forceOCR)
        responseMsg = f"OCR successful! Stored output in '{pathOut}'"
    except ocrmypdf.PriorOcrFoundError as e:
        responseMsg = f"{str(e)}"

    # TODO what do to with the output? => “email” = andere Funkion?

    context.logger.debug("finished ocr")

    # return the encrypted body, and some hard-coded header
    return context.Response(body=responseMsg,
							headers={'x-encrypt-algo': 'aes256'},
							content_type='text/plain',
							status_code=200)
