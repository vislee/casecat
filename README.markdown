Name
====

casecat - The http test case play tool.

Table of Contents
=================

* [Name](#name)
* [Build](#build)
* [Use](#use)
* [Case Example](#case-example)
* [Author](#author)
* [Copyright and License](#copyright-and-license)
* [See Also](#see-also)


Build
=====


[Back to TOC](#table-of-contents)

Use
===

```sh

./casecat -h
Name
  ./casecat - Play testcase or proxy tool (version:1.00)
Usage of ./casecat:
  -PPEnable
    	The proxy protocol enable. true or false (default true)
  -addr string
    	Spec service listen addr. 'IP:port'
  -case string
    	Test play case (default "./case.json")
  -cliAddr string
    	The proxy protol client addr. 'IP' (default "127.0.0.1")
  -proxyAddr string
    	The proxy model listen addr. '[IP]:port'
  -vmHostAddr string
    	The proxy protol vm host addr. 'IP[:port]' (default "127.0.0.1")
Examples
	$ ./casecat -addr=127.0.0.1:80 -vmHostAddr=127.0.0.1 -cliAddr=127.0.0.1 -case=test.json
	$ ./casecat -addr=127.0.0.1:80 -vmHostAddr=127.0.0.1:8989 -cliAddr=127.0.0.1 -proxyAddr=:8080
	$ ./casecat --case help
Authors
	vislee


### testcase example
./casecat --case help
$ cat case.json
[
  {
    "title": "",    ### http test case title.
    "delay": 0,     ### the http test case x(s) after play. default:0
    "repeat": 0,    ### the http test case play times. default:1
    "req": {        ### the http test case request.
      "method": "", ### the http test case request method.
      "url": "",    ### the http test case request url.
      "host": "",   ### the http test case request host.
      "headers": {  ### the http test case request header. example: "user-agent": "curl"
        "": ""
      },
      "body": ""    ### the http test case request body.
    },
    "resp": {       ### the http response.
      "status": {   ### the http response status.
        "type": "", ### match type: "equal" - exact match, "contain" - Contains match, "regex" - Regular match.
        "value": "" ### match value.
      },
      "headers": [
        {
          "key": "",
          "type": "",
          "value": ""
        }
      ],
      "body": {
        "type": "",
        "value": ""
      }
    }
  }
]

```

[Back to TOC](#table-of-contents)

Case Example
============

```json
[
    {
        "title": "Test1 get hello world",
        "req": {
            "url": "/hello/world",
            "host": "www.test.com",
            "headers": {
            }
        },
        "resp": {
            "status": {
                "type": "contain",
                "value": "200"
            },
            "headers": [
                {
                    "key": "UUID",
                    "type": "regex",
                    "value": "[a-z,0-9]{32}"
                },
                {
                    "key": "X-Hello",
                    "type": "equal",
                    "value": "world"
                }
            ],
            "body": {
                "type": "equal",
                "value": "hello world!"
            }
        }
    },
    {
        "title": "Test2 Post hello world",
        "req": {
            "method": "POST",
            "url": "/t",
            "host": "www.test.com",
            "headers": {
                 "x-user": "casecatv1.0"
            },
            "body": "hello world"
        },
        "resp": {
            "status": {
                "type": "contain",
                "value": "200"
            },
            "headers": [
            ]
        }
    }
]
```

[Back to TOC](#table-of-contents)


Author
======

wenqiang li(vislee)

[Back to TOC](#table-of-contents)


Copyright and License
=====================

This module is licensed under the GPLV3 license.

Copyright (C) 2019, by vislee.

All rights reserved.

[Back to TOC](#table-of-contents)


See Also
========

* the [go-httppc](https://github.com/vislee/go-httppc) library.


[Back to TOC](#table-of-contents)

