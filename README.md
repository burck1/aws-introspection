# AWS Introspection

`introspect` is a command line tool and http server meant for allowing you to get metadata information about your AWS infrastructure. 

`introspect` supports generic Windows, Linux, and macOS computers for basic metadata as well as AWS EC2, AWS ECS, and AWS CodeBuild for advanced metadata including [EC2 Instance Metadata](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html), [ECS Task Metadata](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint.html), and [ECS Container Metadata](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/container-metadata.html).

## Building `introspect`

```sh
~$ git clone https://github.com/burck1/aws-introspection.git
~$ cd aws-introspection
~/aws-introspection$ go get github.com/aws/aws-sdk-go/aws github.com/aws/aws-sdk-go/aws/ec2metadata github.com/aws/aws-sdk-go/aws/session github.com/matishsiao/goInfo
~/aws-introspection$ go build -o introspect main.go
```

## Running `introspect`

```sh
$ ./introspect -h
Usage of introspect:
  -c    output compact json
  -port string
        the port to host the server at (default "42011")
  -s    host as a http server
```

```sh
$ ./introspect
{
  "startTime": "...",
  "requestTime": "...",
  "hostname": "...",
  "user": { ... },
  "group": { ... },
  "system": { ... },
  "env": { ... },
  "ec2InstanceMetadata": { ... },
  "ecsContainerMetadata": { ... },
  "ecsContainerStats": { ... },
  "ecsTaskMetadata": { ... },
  "ecsTaskStats": { ... },
  "ecsContainerMetadataFile": { ... }
}
```

```sh
$ ./introspect -s
2019/01/15 23:43:22 Server listening on 0.0.0.0:42011
# Visit http://localhost:42011 in your browser
2019/01/15 23:43:34 GET /
```
