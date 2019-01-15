package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"time"
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
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "")
		err := encoder.Encode(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		w.Header().Set("Allow", "GET")
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
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
	StartTime   string `json:"startTime"`
	RequestTime string `json:"requestTime"`
}

func introspect() introspection {
	now := utcNow()

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
	}
}
