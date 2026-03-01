#!/bin/bash
HOST=192.168.50.245
ssh $HOST 'mkdir ~/gomon/'
scp src/bin/gomon_amd64 config.yaml $HOST:/home/$USER/gomon
