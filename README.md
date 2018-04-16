jagozzi [![Build Status](https://travis-ci.org/rbeuque74/jagozzi.png?branch=master)](https://travis-ci.org/rbeuque74/jagozzi) [![GitHub release](https://img.shields.io/github/release/rbeuque74/jagozzi.svg)](https://github.com/rbeuque74/jagozzi/releases)
==============================

jagozzi is a light monitoring daemon for severals service in order to report results checks to a remote NSCA server.

This program is a Golang clone of [sauna](https://github.com/NicolasLM/sauna) that will parse the same configuration file format.

Services included
-----------------

- Supervisor
- Command
- Processes
- HTTP
- Marathon

Installation
------------

jagozzi can be installed using this command:

```
go install github.com/rbeuque74/jagozzi
```

License
-------

MIT, see [LICENSE](LICENSE)
