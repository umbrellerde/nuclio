#! /usr/bin/env python3

import signal
import subprocess
import os
import sys
import time
import platform

# function that runs a docker container based on an image and commands using subprocess
# commands can be multiple parameters in one string that are split by spaces
# returns the container id
def run_container(image, commands):
    # Delete the container if it already exists
    subprocess.run(["docker", "rm" "load_generator"], stdout=subprocess.PIPE)
    # run the container
    # generate arguments list by splitting commands up into a list
    container = subprocess.run(["docker", "run", "--name", "load_generator", "-d", image] + commands.split(" "), stdout=subprocess.PIPE)

    # get the container id
    container_id = container.stdout.decode('utf-8').strip()
    return container_id

# function that builds a docker image out of the dockerfile in the given path
# returns the image id
def build_image(path):
    # build the image
    image = subprocess.run(["docker", "build", "-t", "load_generator", path], stdout=subprocess.PIPE)
    # get the image id
    image_id = image.stdout.decode('utf-8').strip()
    return image_id

# function that updates the available cpus of a docker container
def update_cpus(container_id, cpus):
    # update the cpus, ignoring the output
    subprocess.run(["docker", "update", "--cpus", cpus, container_id], stdout=subprocess.DEVNULL)

# function that stops a docker container
def stop_container(container_id):
    # stop the container, ignoring the output
    print("Stopping container " + container_id)
    subprocess.run(["docker", "stop", container_id], stdout=subprocess.DEVNULL)
    subprocess.run(["docker", "rm", container_id], stdout=subprocess.DEVNULL)

# function that gets three parameters: the experiment start, the end, and the current time.
# depending on this, it wil output the cpu share that should be avaliable.
# the cpu share should be high% during the first third of the experiment, low% during the last third, and scale linearly in between
def get_cpu_share(start, end, current):
    # get the length of the experiment
    length = end - start
    # get the current time in the experiment
    current = current - start

    # if the current time is in the first third of the experiment
    if current < length/3:
        return "0.80"
    # if the current time is in the last third of the experiment
    elif current > 2*length/3:
        return "0.15"
    # if the current time is in the middle third of the experiment
    else:
        # calculate the cpu share based on the current time
        return str(round(((-1.95/length)*current + 1.45), 2))
        # the value if you want to go from 50 to 15:
        # return str(round(((-105/length)*current + 85)/100, 2))

if __name__ == "__main__":

    # if no parameters are passed, print a help text explaining the usage
    if len(sys.argv) < 2:
        print("Usage: python3 generate_load.py <duration in minutes of test>")
        print("This will create a new docker container that is under a varying amount of load for the given duration.")
        sys.exit(0)

    build_image("../CpuLoadGenerator")
    id = run_container("load_generator", "--cpu 0")
    print("Container id: " + id)

    # automatically stop the container if the script is cancelled
    def signal_handler(sig, frame):
        print("Interrupt received")
        stop_container(id)
        sys.exit(0)
    signal.signal(signal.SIGINT, signal_handler)

    # get the start time as current time, get experiment duration in minutes from command line, and calculate end time based on that
    start = time.time()
    duration = int(sys.argv[1])
    end = start + duration * 60

    cpu_count = 5 if platform.system() == "Darwin" else os.cpu_count()
    # set the cpus of the load_generator container to get_cpu_share for the duration of the experiment
    while time.time() < end:
        curr_time = time.time()
        share = get_cpu_share(start, end, curr_time)
        # to get the numbers of cpu that should be available, multiply the share with the number of cpus on the machine
        share_all_cpus = str(round(float(share) * cpu_count, 2))
        update_cpus(id, share_all_cpus)
        # on every ~10th iteration, print the current cpu share
        if int(curr_time) % 10 == 0:
            print("Current cpu share:", share)
            print("Passed to docker:", share_all_cpus)
        time.sleep(1)

    # stop the container in the end
    stop_container(id)
    