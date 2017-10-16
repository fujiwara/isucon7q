#!/bin/sh

nginx-build -verbose -openresty -openrestyversion 1.11.2.1 -d openresty -c ./configure.sh
