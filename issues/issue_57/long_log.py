#! /usr/bin/env python3

import datetime

cntr=0
line = "="
while True:
    print("long line number {0}: {1}".format(cntr, line))
    cntr += 1
    line += line

