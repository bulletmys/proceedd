#!/bin/bash

git clone https://github.com/bulletmys/proceedd.git

cd proceedd

sudo apt update

sudo apt install -y golang-1.16-go

alias go="/usr/lib/go-1.16/bin/go"
