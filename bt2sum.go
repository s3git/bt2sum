// bt2sum command calculates BLAKE2 Tree hashing checksums in 'unlimited fanout' mode for files.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/dchest/blake2s"
	"github.com/minio/blake2b-simd"
)

var (
	algoFlag = flag.String("a", "blake2b", "Hash algorithm (blake2b, blake2s)")
	sizeFlag = flag.Int("s", 0, "Digest size in bytes (0 defaults to max for algorithm)")
	cpu      = flag.Int("cpus", runtime.NumCPU(), "Number of CPUs to use. Defaults to number of processors.")
	tree     = flag.String("t", "5m", "Chunk size in bytes for tree mode . If size ends with a 'k', 'm', or 'g', it is multiplied by 1024 (1K), 1048576 (1M), or 1073741824 (1G)")
	leafSize = uint(0)
)

func getMultiplier(spec string) uint {
	m := uint(strings.IndexAny("kmg", strings.ToLower(spec)))
	return 1 << ((m + 1) * 10)
}

func parseSize(input string) uint {
	re := regexp.MustCompile("\\d+")
	m := re.FindAllStringSubmatchIndex(input, -1)
	if len(m) != 1 {
		return uint(0)
	}
	i, err := strconv.Atoi(input[m[0][0]:m[0][1]])
	if err != nil {
		return uint(0)
	}
	mult := getMultiplier(input[m[0][1]:])

	return uint(i) * mult
}

type hashDesc struct {
	name    string
	maxSize int
	maker   func(size uint8, offset uint64, lastNode bool, depth byte, leafSize uint32) (hash.Hash, error)
}

var algorithms = map[string]hashDesc{
	"blake2b": {
		"BLAKE2b",
		blake2b.Size,
		func(size uint8, offset uint64, lastNode bool, depth byte, leafSize uint32) (hash.Hash, error) {
			return blake2b.New(&blake2b.Config{
				Size: size,
				Tree: &blake2b.Tree{
					Fanout:        0,
					MaxDepth:      2,
					LeafSize:      leafSize, // Leaf maximal byte length (4 bytes)
					NodeOffset:    offset,
					NodeDepth:     depth,
					InnerHashSize: size,
					IsLastNode:    lastNode,
				},
			})
		},
	},
	"blake2s": {
		"BLAKE2s",
		blake2s.Size,
		func(size uint8, offset uint64, lastNode bool, depth byte, leafSize uint32) (hash.Hash, error) {
			return blake2s.New(&blake2s.Config{
				Size: size,
				Tree: &blake2s.Tree{
					Fanout:        0,
					MaxDepth:      2,
					LeafSize:      leafSize, // Leaf maximal byte length (4 bytes)
					NodeOffset:    offset,
					NodeDepth:     depth,
					InnerHashSize: size,
					IsLastNode:    lastNode,
				},
			})
		},
	},
}

// Worker routine for computing hash for a chunk
func calcChunkWorkers(algo hashDesc, chunks <-chan chunkInput, results chan<- chunkOutput) {

	for c := range chunks {

		blake, err := algo.maker(uint8(*sizeFlag), uint64(c.part), c.lastChunk, 0, uint32(c.leafSize))
		if err != nil {
			fmt.Println("Failing to create algorithm: ", err)
			return
		}

		blake.Reset()
		_, err = io.Copy(blake, bytes.NewBuffer(c.partBuffer))
		if err != nil {
			fmt.Println("Failing to compute hash: ", err)
			results <- chunkOutput{digest: []byte(""), part: c.part}
		} else {
			digest := blake.Sum(nil)
			results <- chunkOutput{digest: digest, part: c.part}
		}
	}
}

type chunkInput struct {
	part       int
	partBuffer []byte
	lastChunk  bool
	leafSize   uint
	level      int
}

type chunkOutput struct {
	digest []byte
	part   int
}

func calcStream(algo hashDesc, r io.Reader, fileSize int64) (digest []byte, err error) {

	var wg sync.WaitGroup
	chunks := make(chan chunkInput)
	results := make(chan chunkOutput)

	// Start one go routine per CPU
	for i := 0; i < *cpu; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			calcChunkWorkers(algo, chunks, results)
		}()
	}

	// Push chunks onto input channel
	go func() {
		for part, totalSize := 0, int64(0); ; part++ {
			partBuffer := make([]byte, leafSize)
			n, err := r.Read(partBuffer)
			if err != nil {
				return
			}
			partBuffer = partBuffer[:n]

			totalSize += int64(n)
			lastChunk := uint(n) < leafSize || uint(n) == leafSize && totalSize == fileSize

			chunks <- chunkInput{part: part, partBuffer: partBuffer, lastChunk: lastChunk, leafSize: leafSize, level: 0}

			if lastChunk {
				break
			}
		}

		// Close input channel
		close(chunks)
	}()

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(results) // Close output channel
	}()

	// Create hash based on chunk number with digest of chunk
	// (number of chunks upfront is unknown for stdin stream)
	digestHash := make(map[int][]byte)
	for r := range results {
		digestHash[r.part] = r.digest
	}

	// Concatenate digests of chunks
	b := make([]byte, len(digestHash)**sizeFlag)
	for index, val := range digestHash {
		offset := *sizeFlag * index
		copy(b[offset:offset+*sizeFlag], val[:])
	}

	rootBlake, err := algo.maker(uint8(*sizeFlag), 0, true, 1, uint32(leafSize))
	if err != nil {
		return nil, err
	}

	// Compute top level digest
	rootBlake.Reset()
	_, err = io.Copy(rootBlake, bytes.NewBuffer(b))
	digest = rootBlake.Sum(nil)

	return digest, nil
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(*cpu)
	if *tree != "" {
		leafSize = parseSize(*tree)
		if leafSize == 0 {
			flag.Usage()
			fmt.Fprintf(os.Stderr, "bad tree size: %s\n", *tree)
			os.Exit(1)
		}
	}

	algo, ok := algorithms[*algoFlag]
	if !ok {
		flag.Usage()
		fmt.Fprintf(os.Stderr, `unsupported algorithm: %s`, *algoFlag)
		os.Exit(1)
	}
	if *sizeFlag == 0 {
		*sizeFlag = algo.maxSize
	} else if *sizeFlag > algo.maxSize {
		fmt.Fprintf(os.Stderr, "error: size too large")
		os.Exit(1)
	}

	if flag.NArg() == 0 {
		// Read from stdin.
		digest, err := calcStream(algo, os.Stdin, -1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s-%d = %x\n", algo.name, *sizeFlag, digest)
		os.Exit(0)
	}
	exitNo := 0
	for i := 0; i < flag.NArg(); i++ {
		filename := flag.Arg(i)
		f, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "(%s) %s\n", filename, err)
			exitNo = 1
			continue
		}
		defer f.Close()
		fileInfo, err := f.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "(%s) %s\n", filename, err)
			exitNo = 1
			continue
		}
		fileSize := fileInfo.Size()

		digest, err := calcStream(algo, f, fileSize)
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "(%s) %s\n", filename, err)
			exitNo = 1
			continue
		}
		fmt.Printf("%s-%d (%s) = %x\n", algo.name, *sizeFlag, filename, digest)
	}
	os.Exit(exitNo)
}
