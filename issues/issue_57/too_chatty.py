#! /usr/bin/env python3

import datetime

start = datetime.datetime.now()
cntr=0
duration=""
while True:
    print("I am long line number {0} ====================================== Duration per 100k: {1}".format(cntr, duration))
    cntr += 1
    if cntr % 100000 == 0:
      duration = datetime.datetime.now() - start
      start = datetime.datetime.now()

