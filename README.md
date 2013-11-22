GO-Meter
=========

HTTP benchmark tool

Usage
----

```
$ ./go-meter -t 12 -c 400 -d 30s -u http://localhost/index.html -v
```

- `-t` Threads count
- `-c` TCP open connection count
- `-d` Test duration, example `30s`, `1m`, `1m30s`
- `-reconnect` Reconnect on every request
- `-m` HTTP method: `GET`/`POST`/`PUT`/`DELETE`
- `-es` Exclude first seconds from stats aggregation, use for wake up http server,  example `3s`, `5s`
- `-mrq` Max request count per second, `-1` for unlimit
- `-u` URL for testing
- `-v` View statistic in runtime
- `-s` Source file with `\n` delimeter for `POST`/`PUT` requests or list of URLs for `GET`/`DELETE`


Source file example:

`POST`/`PUT` 

```
{req: 1}
{req: 2}
{req: 3}
```

`GET`/`DELETE` 

```
http://localhost/index.html
http://localhost/page1/sub1
http://localhost/page1/sub2?rnd=22
```