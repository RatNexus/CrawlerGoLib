#!/bin/bash

ls -1 | grep -v "$(basename "$0")" | tail -n 1 | xargs cat
