package main

import (
	"sort"
	"strconv"
	"sync"
)

func CombineResults(in, out chan interface{}) {
	results := make([]string, 0)
	for inputData := range in {
		data := inputData.(string)
		results = append(results, data)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i] < results[j]
	})

	var total string
	for i, str := range results {
		total += str
		if i != len(results)-1 {
			total += "_"
		}
	}
	out <- total
}

type OrderString struct {
	index int
	hash  string
}

const multiParam int = 6 // number of hashes in MultiHash

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for inputData := range in {
		wg.Add(1)

		// MultiHash implementation
		// (DataSignerCrc32)x6: 1 s (parallel)
		// Total: ~1 s
		go func(data string, wg *sync.WaitGroup) {
			defer wg.Done()

			resCRC32 := make(chan OrderString, multiParam)
			for i := 0; i < multiParam; i++ {
				go func(num int) {
					resCRC32 <- OrderString{num, DataSignerCrc32(strconv.Itoa(num) + data)}
				}(i)
			}

			var multiHash string
			tmpSlice := make([]string, multiParam) // maintaining order
			for i := 0; i < multiParam; i++ {
				val := <-resCRC32
				tmpSlice[val.index] = val.hash
			}
			for _, str := range tmpSlice {
				multiHash += str
			}
			out <- multiHash
		}(inputData.(string), wg)
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	var mux sync.Mutex

	wg := &sync.WaitGroup{}
	for inputData := range in {
		wg.Add(1)

		// SingleHash implementation
		// 1) DataSignerCrc32: 1 s (parallel)
		// 2) DataSignerMd5: 10 ms
		// 3) DaraSignerCrc32: 1 s
		// Total: ~1.1 s
		go func(data string, wg *sync.WaitGroup) {
			defer wg.Done()

			resCRC32 := make(chan string)
			go func() {
				resCRC32 <- DataSignerCrc32(data)
			}()
			mux.Lock()
			dataMD5 := DataSignerMd5(data)
			mux.Unlock()
			dataCRC32_MD5 := DataSignerCrc32(dataMD5)
			dataCRC32 := <-resCRC32
			out <- (dataCRC32 + "~" + dataCRC32_MD5)
		}(strconv.Itoa(inputData.(int)), wg)
	}
	wg.Wait()
}

type ReadWrite struct {
	in  chan interface{}
	out chan interface{}
}

func initFd(size int) []ReadWrite {
	fd := make([]ReadWrite, size)
	fd[0] = ReadWrite{make(chan interface{}, MaxInputDataLen), make(chan interface{}, MaxInputDataLen)}
	for i := 1; i < size; i++ {
		fd[i] = ReadWrite{fd[i-1].out, make(chan interface{}, MaxInputDataLen)}
	}
	return fd
}

func ExecutePipeline(jobs ...job) {
	fd := initFd(len(jobs))

	for i := 0; i < len(jobs); i++ {
		worker := jobs[i]
		go func(worker job, fd ReadWrite) {
			worker(fd.in, fd.out)
			close(fd.out)
		}(worker, fd[i])
	}

	// waiting for the closing of the last channel
	for _ = range fd[len(jobs)-1].out {
	}
}
