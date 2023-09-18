# Experiment Setup

1. Start Minio, Postgres, Nuclio (follow all steps in main readme for profaastinate). Don't forget `docker network create profaastinate`
2. Create a Minio Bucket "profaastinate" with the files "fusionize.pdf" and "fusionizeOCR.pdf". These files will be used for all checks etc.
3. Create the Functions in Nuclio, one function per folder in `usecase`. Make sure to copy over the requirements to the build instructions and to create a trigger with enough workers

# Experiment Run

0. Start the collectCPUusage.sh script
1. Start CPU Load Generator and K6 load generator
2. Wait until all functions have been executed (i.e., db has no entries anymore)
3. stop collecting CPU usage and copy over the resulting file
4. `./collectLogsFromContainers.sh` and copy over the resulting files