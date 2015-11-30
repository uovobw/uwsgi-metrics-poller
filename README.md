Uwsgi Metrics Poller
====================

This small hack watches an [Etcd](https://github.com/coreos/etcd) directory and fetches all the keys
in there, assuming they are in the form HOST:PORT, and polls the host on a given port (not
necessarily the same as the PORT above) for the [uWSGI stats server](https://uwsgi-docs.readthedocs.org/en/latest/StatsServer.html)

It then pushes the collected metrics on [AWS Cloudwatch](https://aws.amazon.com/cloudwatch/) for alarming and autoscaling

This code is very very unstable and only fits my use case, so be warned

Dependencies
============

- [AWS Golang SDK](github.com/aws/aws-sdk-go)
- [Kingping flag parsing](http://github.com/alecthomas/kingpin)
- [@Fatih's Set](https://github.com/fatih/set)
- [Etcd client library](github.com/coreos/etcd/client)
- [Golang context library](golang.org/x/net/context)

there is currently no package management nor vendorization

Installation
============

The command is not `go get`-table since the current process builds it locally. A simple

'''
git clone https://github.com/uovobw/uwsgi-metrics-poller
cd uwsgi-metrics-poller
go build
'''

should suffice to get a runnable executable

Configuration
=============

There is currently no configuration file (or environment variable) support, but it will probably come in the future

Run with `-h` to get a summary of the arguments and their default values
