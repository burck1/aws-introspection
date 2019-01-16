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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/matishsiao/goInfo"
)

var startTime string
var metadata *ec2metadata.EC2Metadata

func main() {
	startTime = utcNow()

	serverPtr := flag.Bool("s", false, "host as a http server")
	portPtr := flag.String("port", "42011", "the port to host the server at")
	compactPtr := flag.Bool("c", false, "output compact json")
	flag.Parse()

	metadata = newMetadataClient()

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
		writeIntrospection(*compactPtr)
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

func newMetadataClient() *ec2metadata.EC2Metadata {
	awsConfig := aws.NewConfig()
	metadataSession, err := session.NewSession(awsConfig)
	if err != nil {
		log.Fatal("session.NewSession: ", err)
	}
	return ec2metadata.New(metadataSession)
}

func writeIntrospection(compact bool) {
	data := introspect()
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if compact {
		encoder.SetIndent("", "")
	} else {
		encoder.SetIndent("", "  ")
	}
	err := encoder.Encode(data)
	if err != nil {
		log.Fatal("MarshalIndent:", err)
	}
}

func httpGet(client *http.Client, endpoint string) io.ReadCloser {
	resp, err := client.Get(endpoint)
	if err != nil {
		log.Fatal("GET "+endpoint+": ", err)
	}
	if resp.StatusCode != http.StatusOK {
		if resp.Body != nil {
			defer close(resp.Body)
		}
		log.Fatal("GET " + endpoint + " > " + string(resp.StatusCode) + " " + resp.Status)
	}
	return resp.Body
}

// func httpGetString(client *http.Client, endpoint string) string {
// 	body := httpGet(client, endpoint)
// 	if body == nil {
// 		return ""
// 	}
// 	defer close(body)
// 	data, err := ioutil.ReadAll(body)
// 	if err != nil {
// 		log.Fatal("ioutil.ReadAll: ", err)
// 	}
// 	return string(data)
// }

func httpGetJSON(client *http.Client, endpoint string) map[string]interface{} {
	body := httpGet(client, endpoint)
	if body == nil {
		return nil
	}
	defer close(body)
	var data map[string]interface{}
	decoder := json.NewDecoder(body)
	err := decoder.Decode(&data)
	if err != nil {
		log.Fatal("json.NewDecoder.Decode: ", err)
	}
	return data
}

func getContainerIDFromtaskMetadataJSON(taskMetadata map[string]interface{}) string {
	// this function gets the DockerId of the first container listed
	// in the task metadata endpoint where the Type == NORMAL
	// this same logic is used by the taskmetadata-validator in github.com/aws/amazon-ecs-agent
	// https://github.com/aws/amazon-ecs-agent/blob/master/misc/taskmetadata-validator/taskmetadata-validator.go
	/*
		{
			"Containers": [{
				"Type": "foo",
				"DockerId": "123"
			}, {
				"Type": "NORMAL",
				"DockerId": "456"
			}]
		}
	*/
	containerID := ""
	containersRaw, ok := taskMetadata["Containers"]
	if ok {
		containers, ok := containersRaw.([]interface{})
		if ok {
			for _, containerRaw := range containers {
				container, ok := containerRaw.(map[string]interface{})
				if ok {
					containerTypeRaw, ok := container["Type"]
					if ok {
						containerType, ok := containerTypeRaw.(string)
						if ok {
							if containerType == "NORMAL" {
								containerIDRaw, ok := container["DockerId"]
								if ok {
									containerID, ok = containerIDRaw.(string)
									if ok {
										break
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return containerID
}

type introspection struct {
	StartTime                string                                   `json:"startTime"`
	RequestTime              string                                   `json:"requestTime"`
	Hostname                 string                                   `json:"hostname"`
	User                     *user.User                               `json:"user"`
	Group                    *user.Group                              `json:"group"`
	System                   *goInfo.GoInfoObject                     `json:"system"`
	Env                      map[string]string                        `json:"env"`
	EC2InstanceMetadata      *ec2metadata.EC2InstanceIdentityDocument `json:"ec2InstanceMetadata"`
	ECSContainerMetadata     map[string]interface{}                   `json:"ecsContainerMetadata"`
	ECSContainerStats        map[string]interface{}                   `json:"ecsContainerStats"`
	ECSTaskMetadata          map[string]interface{}                   `json:"ecsTaskMetadata"`
	ECSTaskStats             map[string]interface{}                   `json:"ecsTaskStats"`
	ECSContainerMetadataFile map[string]interface{}                   `json:"ecsContainerMetadataFile"`
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
	var ec2 *ec2metadata.EC2InstanceIdentityDocument
	if metadata.Available() {
		iid, err := metadata.GetInstanceIdentityDocument()
		if err != nil {
			log.Fatal("metadata.GetInstanceIdentityDocument: ", err)
		}
		ec2 = &iid
	}

	// ECS: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint.html
	// https://github.com/aws/amazon-ecs-agent/blob/master/misc/v3-task-endpoint-validator/v3-task-endpoint-validator.go
	var hasContainerMetadataPath, hasExecutionEnv bool
	var executionEnv, containerMetadataPath, containerStatsPath, taskMetadataPath, taskStatsPath string
	var containerMetadata, containerStats, taskMetadata, taskStats map[string]interface{}
	containerMetadataPath, hasContainerMetadataPath = os.LookupEnv("ECS_CONTAINER_METADATA_URI")
	executionEnv, hasExecutionEnv = os.LookupEnv("AWS_EXECUTION_ENV")
	if hasContainerMetadataPath {
		// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v3.html
		containerStatsPath = containerMetadataPath + "/stats"
		taskMetadataPath = containerMetadataPath + "/task"
		taskStatsPath = taskMetadataPath + "/stats"

		client := &http.Client{
			Timeout: 5 * time.Second,
		}
		containerMetadata = httpGetJSON(client, containerMetadataPath)
		containerStats = httpGetJSON(client, containerStatsPath)
		taskMetadata = httpGetJSON(client, taskMetadataPath)
		taskStats = httpGetJSON(client, taskStatsPath)
	} else if hasExecutionEnv && executionEnv == "AWS_ECS_EC2" {
		// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v2.html
		taskMetadataPath = "http://169.254.170.2/v2/metadata"
		taskStatsPath = "http://169.254.170.2/v2/stats"

		client := &http.Client{
			Timeout: 5 * time.Second,
		}
		taskMetadata = httpGetJSON(client, taskMetadataPath)
		taskStats = httpGetJSON(client, taskStatsPath)

		containerID := getContainerIDFromtaskMetadataJSON(taskMetadata)
		if containerID != "" {
			containerMetadataPath = taskMetadataPath + "/" + containerID
			containerStatsPath = taskStatsPath + "/" + containerID

			containerMetadata = httpGetJSON(client, containerMetadataPath)
			containerStats = httpGetJSON(client, containerStatsPath)
		}
	}

	// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/container-metadata.html
	var containerMetadataFile map[string]interface{}
	metadataFile, hasMetadataFile := os.LookupEnv("ECS_CONTAINER_METADATA_FILE")
	if hasMetadataFile {
		f, err := os.Open(metadataFile)
		if err == nil {
			defer close(f)
			decoder := json.NewDecoder(f)
			err := decoder.Decode(&containerMetadataFile)
			if err != nil {
				log.Fatal("json.NewDecoder.Decode: ", err)
			}
		}
	}

	return introspection{
		StartTime:                startTime,
		RequestTime:              now,
		Hostname:                 hostname,
		User:                     currentUser,
		Group:                    primaryGroup,
		System:                   system,
		Env:                      env,
		EC2InstanceMetadata:      ec2,
		ECSContainerMetadata:     containerMetadata,
		ECSContainerStats:        containerStats,
		ECSTaskMetadata:          taskMetadata,
		ECSTaskStats:             taskStats,
		ECSContainerMetadataFile: containerMetadataFile,
	}
}
