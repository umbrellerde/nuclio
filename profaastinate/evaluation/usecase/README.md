4 functions.

First one receives a pdf file and checks it's metadata and file size

Second one checks the pdf for viruses (by reading it in and seeing if anything goes wrong :) )

Third one performs OCR on the pdf and creates an annotated pdf

Fourth one gets the text from the pdf and sends it via mail (just print it to sysout)

# Experiment Setup

1. Start Minio, Postgres, Nuclio (follow all steps in main readme for profaastinate)
2. Create a Minio Bucket "profaastinate" with a file "test.pdf". This file will be used for all checks etc.
3. Create the Functions in Nuclio, one function per folder in `usecase`. Make sure to copy over the requirements to the buld

## places to fix networking

* ⌘⇧F: "host.docker.internal" -> localhost