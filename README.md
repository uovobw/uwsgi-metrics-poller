Uwsgi Metrics Poller
====================

This small hack watches an [Etcd](https://github.com/coreos/etcd) directory and fetches all the keys
in there, assuming they are in the form <host>:<port>, and polls the host on a given port (not
necessarily the same as the <port> above) for the [uWSGI stats server](https://uwsgi-docs.readthedocs.org/en/latest/StatsServer.html)
It then pushes the collected metrics on [AWS Cloudwatch](https://aws.amazon.com/cloudwatch/) for alarming and autoscaling

This code is very very unstable and only fits my use case, so be warned.

Installation
============

TBD

Configuration
=============

There is currently no configuration file (or environment variable) support, but it will probably come in the future
