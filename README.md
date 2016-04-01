BT2SUM
======

This utility is derived from [b2sum](https://bitbucket.org/dchest/b2sum) as developed by Dmitry Chestnykh. It is adapted to use the BLAKE2 Tree hashing mode in so called 'unlimited fanout' mode, as it is used in [s3git](https://github.com/s3git/s3git) and described [here](https://github.com/s3git/s3git/blob/master/BLAKE2.md#blake2-tree-modeunlimited-fanout).

It computes the hashes for the chunks at the leaf level in parallel using by default the number of processors of your computer which gives a nice speed up. Once the hashes for all leaves are available the final hash is computed at level 1.

Build from source
-----------------

To install from source:

```sh
$ go get -d github.com/s3git/bt2sum
$ cd $GOPATH/src/github.com/s3git/bt2sum 
$ go install
$ bt2sum -h
```

Usage
-----

```
Usage of bt2sum:
  -a string
    	Hash algorithm (blake2b, blake2s) (default "blake2b")
  -cpus int
    	Number of CPUs to use. Defaults to number of processors. (default 8)
  -s int
    	Digest size in bytes (0 defaults to max for algorithm)
  -t string
    	Chunk size in bytes for tree mode (defaults to 5M). If size ends with a 'k', 'm', or 'g', it is multiplied by 1024 (1K), 1048576 (1M), or 1073741824 (1G) (default "5m")
```

If no filenames are specified, it reads from standard input.

Examples
--------

```sh
$ echo "hello s3git" | bt2sum
BLAKE2b-64 = 18e622875a89cede0d7019b2c8afecf8928c21eac18ec51e38a8e6b829b82c3ef306dec34227929fa77b1c7c329b3d4e50ed9e72dc4dc885be0932d3f28d7053
$
$ # Output 40 byte checksum
$ echo "hello s3git" | bt2sum -s 40
BLAKE2b-40 = 919f330a1b4a3a02aced735e7675905c159b99e07e0c8aa087d0327b26e4d3aa8323bc82962b8e8e
$
$ # Compute sum for go installer
$ bt2sum go1.6.darwin-amd64.pkg
BLAKE2b-64 (go1.6.darwin-amd64.pkg) = 9be020e41e6fefec6b52b1ae1623a1fdd800c2a5c98d1079c9363107d362fbd558b4e3abb9500ab5f30de9ac708e53ff6b44b1c041edb81cd5df4e29f5dc4e99
$
$ # Now use 10 MB chunks
$ bt2sum -t 10M go1.6.darwin-amd64.pkg
BLAKE2b-64 (go1.6.darwin-amd64.pkg) = bf05d62548d4aeec8eae124dddefe6572482fe1693a252d01adeb0a3b8cfc308860b7e323c1cf1d14ae67542f146667e009be45313e801a952a8da702ec545a9
$
$ # Use blake2s algo
$ bt2sum -a blake2s go1.6.darwin-amd64.pkg
BLAKE2s-32 (go1.6.darwin-amd64.pkg) = 75cbcfafa371ed2afb0f4abce06af44a1261376ee071cd35e698f3f590ace529
```
