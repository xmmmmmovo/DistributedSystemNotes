package mr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	// 这里的mapf和reducef是maph函数和reduce函数

	// Your worker implementation here.
	workerId, nReduce, ok := callRegisterWorker()
	if !ok {
		fmt.Println("获取Id失败！")
		return
	}
	fmt.Println("获取到当前workerId", workerId)

	// uncomment to send the Example RPC to the master.
	// CallExample()
	for {
		log.Println("运行中")
		status, filenames, ok := callFetchWorker()
		if !ok {
			fmt.Println("获取任务失败！")
			return
		}
		fmt.Println("获取状态:", status, "文件列表", filenames)
		switch status {
		case 1:
			err := mapFuncTask(workerId, filenames[0], mapf, nReduce)
			if err != nil {
				fmt.Println("执行map失败！")
				return
			}
		case 2:
			reduceFuncTask(reducef)
		}
		time.Sleep(time.Second * 1)
	}
}

func callRegisterWorker() (int, int, bool) {
	args := RegisterArgs{}
	reply := RegisterReply{}

	err := call("Master.RegisterWorker", &args, &reply)
	return reply.Id, reply.NReduce, err
}

func callFetchWorker() (int, []string, bool) {
	args := FetchArgs{}
	reply := FetchReply{}

	err := call("Master.FetchWorker", &args, &reply)
	return reply.Status, reply.FileNames, err
}

func mapFuncTask(id int, filename string, mapf func(string, string) []KeyValue, nReduce int) error {
	// 读文件
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	intermediate := make(map[int][]KeyValue)
	// map 函数
	mapKV := mapf(filename, string(content))
	for _, v := range mapKV {
		idx := ihash(v.Key) % nReduce
		intermediate[idx] = append(intermediate[idx], v)
	}

	// 写中间文件
	dir := "./tmp/"
	for k, v := range intermediate {
		filename = fmt.Sprintf("mr-%d-%d", id, k)
		f, err := os.Create(dir + filename)
		if err != nil {
			return err
		}
		jsonEncoder := json.NewEncoder(f)
		for _, kv := range v {

			if err := jsonEncoder.Encode(kv); err != nil {
				return err
			}
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	return nil
}

func reduceFuncTask(reducef func(string, []string) string) {
}

//
// send an RPC request to the master, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := masterSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
