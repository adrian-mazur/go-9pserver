go-9pserver
=====
A user space file server implementing [9P2000](https://en.wikipedia.org/wiki/9P_(protocol)) protocol written in Go programming language.
## Building
```
go build .
```
## Example usage
In order to start the server serving files in `/tmp/9p` directory the following command can be used:
```
./9pserver /tmp/9p
```
To mount the file system in Linux at `/mnt/mountdir`:
```
mount -t 9p 127.0.0.1 -o noextend /mnt/mountdir
```
