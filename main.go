package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/matishsiao/goInfo"
)

var startTime string

func main() {
	startTime = utcNow()

	serverPtr := flag.Bool("server", false, "host as a http server")
	portPtr := flag.String("port", "8080", "the port to host the server at")

	flag.Parse()

	if *serverPtr {
		// setup as server
		http.HandleFunc("/", getIntrospection)

		log.Println("Server listening on 0.0.0.0:" + *portPtr)
		err := http.ListenAndServe("0.0.0.0:"+*portPtr, nil)
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	} else {
		// output to command line
		writeIntrospection()
	}
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func getIntrospection(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL)

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := introspect()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		var encoder *json.Encoder

		acceptEncoding := r.Header.Get("Accept-Encoding")
		if strings.Contains(acceptEncoding, "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			// Get a Writer from the Pool
			gzw := takeZipper(w)
			// When done, put the Writer back in to the Pool
			defer returnZipper(gzw)
			defer close(gzw)
			encoder = json.NewEncoder(gzw)
		} else {
			encoder = json.NewEncoder(w)
		}

		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "")
		err := encoder.Encode(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		w.Header().Set("Allow", "GET")
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
	}
}

// use a sync.Pool so we can re-use Writers between goroutines
var zippersPool sync.Pool

func takeZipper(w io.Writer) *gzip.Writer {
	if z := zippersPool.Get(); z != nil {
		zipper := z.(*gzip.Writer)
		zipper.Reset(w)
		return zipper
	}
	return gzip.NewWriter(w)
}

func returnZipper(zipper *gzip.Writer) {
	zippersPool.Put(zipper)
}

func close(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func writeIntrospection() {
	data := introspect()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(data)
	if err != nil {
		log.Fatal("MarshalIndent:", err)
	}
}

type introspection struct {
	StartTime   string               `json:"startTime"`
	RequestTime string               `json:"requestTime"`
	Hostname    string               `json:"hostname"`
	User        *user.User           `json:"user"`
	Group       *user.Group          `json:"group"`
	System      *goInfo.GoInfoObject `json:"system"`
	Env         map[string]string    `json:"env"`
}

func introspect() introspection {
	now := utcNow()

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	currentUser, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	primaryGroup, err := user.LookupGroupId(currentUser.Gid)
	if err != nil {
		log.Fatal(err)
	}

	system := goInfo.GetInfo()

	env := make(map[string]string)
	for _, item := range os.Environ() {
		splits := strings.SplitN(item, "=", 2)
		env[splits[0]] = splits[1]
	}

	// EC2: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html
	// GET http://169.254.169.254/latest/dynamic/instance-identity/document
	// {
	// 	"devpayProductCodes" : null,
	// 	"marketplaceProductCodes" : [ "1abc2defghijklm3nopqrs4tu" ],
	// 	"availabilityZone" : "us-west-2b",
	// 	"privateIp" : "10.158.112.84",
	// 	"version" : "2017-09-30",
	// 	"instanceId" : "i-1234567890abcdef0",
	// 	"billingProducts" : null,
	// 	"instanceType" : "t2.micro",
	// 	"accountId" : "123456789012",
	// 	"imageId" : "ami-5fb8c835",
	// 	"pendingTime" : "2016-11-19T16:32:11Z",
	// 	"architecture" : "x86_64",
	// 	"kernelId" : null,
	// 	"ramdiskId" : null,
	// 	"region" : "us-west-2"
	// }

	// ECS: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint.html
	// ${ECS_CONTAINER_METADATA_URI}

	// ECS env vars
	// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-agent-config.html
	// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/container-metadata.html

	return introspection{
		StartTime:   startTime,
		RequestTime: now,
		Hostname:    hostname,
		User:        currentUser,
		Group:       primaryGroup,
		System:      system,
		Env:         env,
	}
}
