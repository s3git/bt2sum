package main

import (
	"bytes"
	"encoding/hex"
	"runtime"
	"testing"
)

// See http://pythonhosted.org/pyblake2/examples.html for simple tree example,
//   adapted here for 'unlimited fanout' with fanout = 0 and max_depth = 2
//
//      10
//     /  \
//    00  01
//
//>>> from pyblake2 import blake2b
//>>> FANOUT = 0	// WAS 2 IN EXAMPLE
//>>> DEPTH = 2
//>>> LEAF_SIZE = 4096
//>>> INNER_SIZE = 64
//>>> buf = bytearray(6000)
//>>> h00 = blake2b(buf[0:LEAF_SIZE], fanout=FANOUT, depth=DEPTH, leaf_size=LEAF_SIZE, inner_size=INNER_SIZE, node_offset=0, node_depth=0, last_node=False)
//>>> h00.hexdigest()
//'69febe8b43deb8cca4375abb2ebe3533555dc430890cfbb5cc044178593e267f5b0cf62186a80c2602fd37652c594777e3be6107e17337b20133d2893867f9ec'
//>>>
//>>> h01 = blake2b(buf[LEAF_SIZE:], fanout=FANOUT, depth=DEPTH, leaf_size=LEAF_SIZE, inner_size=INNER_SIZE, node_offset=1, node_depth=0, last_node=True)
//>>> h01.hexdigest()
//'4114db272c300f811d6174c7ab134cf01bd91bc30be470bbb96047c05b3063cc58c579873cf19ec130f20043ecebeeab26aac54b909f5f5edc12d38704a33e7e'
//>>>
//>>> h10 = blake2b(digest_size=64, fanout=FANOUT, depth=DEPTH, leaf_size=LEAF_SIZE, inner_size=INNER_SIZE, node_offset=0, node_depth=1, last_node=True)
//>>> h10.update(h00.digest())
//>>> h10.update(h01.digest())
//>>> h10.hexdigest()
//'724b5876d9120855dcf220ee7815a697c6559575aa79d9e2b9f63f06c9f22e532fc32bf356ef1ab26ff776ea960edeb31c54769460f87489f59ae8045122705d'

var sumSimpleTree = string("724b5876d9120855dcf220ee7815a697c6559575aa79d9e2b9f63f06c9f22e532fc32bf356ef1ab26ff776ea960edeb31c54769460f87489f59ae8045122705d")

func TestSimpleTree(t *testing.T) {
	b := make([]byte, 6000)
	algo, _ := algorithms["blake2b"]
	*sizeFlag = 64
	leafSize = 4096
	*cpu = runtime.NumCPU()
	runtime.GOMAXPROCS(*cpu)

	sum, _ := calcStream(algo, bytes.NewBuffer(b), -1)
	if hex.EncodeToString(sum) != sumSimpleTree {
		t.Errorf("Got:%s want:%s", hex.EncodeToString(sum), sumSimpleTree)
	}
}
