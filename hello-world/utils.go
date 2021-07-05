package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/fatih/structs"
	"github.com/noirbizarre/gonja"
	"gopkg.in/yaml.v2"
)

// Create struct to hold info about new item
type Item struct {
	Year   int
	Title  string
	Plot   string
	Rating float64
}

// This was used to with pongo2 and no longer needed keeping it for reference

// type MyWriter struct {
// 	S string
// }

// // This method means type T implements the interface I,
// // but we don't need to explicitly declare that it does so.
// func (t MyWriter) Write(p []byte) (n int, err error) {
// 	s := string(p)
// 	fmt.Println(s)
// 	return n, err
// }

// WriteToDynamodb is a function that takes data and writes it to Dynamdb
func WriteToDynamodb() {
	// Initialize a session that the SDK will use to load
	// credentials from the shared credentials file ~/.aws/credentials
	// and region from the shared configuration file ~/.aws/config.
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create DynamoDB client
	svc := dynamodb.New(sess)
	randFloat := (rand.Float64() * 100) / 10
	item := Item{
		Year:   2015,
		Title:  "The Big New Movie " + fmt.Sprintf("%v", randFloat),
		Plot:   "Nothing happens at all.",
		Rating: randFloat,
	}

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		log.Fatalf("Got error marshalling new movie item: %s", err)
	}
	// Create item in table Movies
	tableName := "Movies"

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	}

	_, err = svc.PutItem(input)
	if err != nil {
		log.Fatalf("Got error calling PutItem: %s", err)
	}

	year := strconv.Itoa(item.Year)

	fmt.Println("Successfully added '" + item.Title + "' (" + year + ") to table " + tableName)

}

//GenerateLogs is a function that generates random metrics logs to be processed by cloudwatch
func GenerateLogs(templateName string, context gonja.Context) (out string, err error) {
	var templateBody = gonja.Must(gonja.FromFile("templates/" + templateName))

	// This was used to with pongo2 and no longer needed keeping it for reference
	// var writer io.Writer = MyWriter{}
	rand.Seed(time.Now().UnixNano())

	v := rand.Perm(9000)
	context["Items"] = v
	// var context = gonja.Context{
	// 	"appName":     "modamodApp",
	// 	"version":     "1.0.1.0",
	// 	"appFullName": "My Awesome App",
	// 	"client":      "modamod",
	// 	"env":         "dev",
	// 	"items":       v,
	// }
	out, err = templateBody.Execute(context)
	return out, err
}

//ContextStruct defines the yaml file that we are going to read the context from
type ContextStruct struct {
	AppName     string `yaml:"appName"`
	Version     string `yaml:"version"`
	AppFullName string `yaml:"appFullName"`
	Client      string `yaml:"client"`
	Env         string `yaml:"env"`
}

//ContextFromFile is a function that reads a yaml file and returns a gonja.Context object.
func ContextFromFile(contextFile string) (gonja.Context, error) {
	t := ContextStruct{}
	data, err := ioutil.ReadFile(contextFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &t)
	m := structs.Map(t)
	context := gonja.Context(m)
	return context, err
}

//PutLogEvents is a function that puts a log message to a log group.
func PutLogEvents(cloudwatch *cloudwatchlogs.CloudWatchLogs, logGroupName string, logStreamName string, msg *string) (output *(cloudwatchlogs.PutLogEventsOutput), err error) {

	var logEventsInput cloudwatchlogs.PutLogEventsInput

	nanoTimestamp := time.Now().Unix() * 1000
	logEvent := cloudwatchlogs.InputLogEvent{
		Message:   msg,
		Timestamp: &nanoTimestamp,
	}
	logEventsInput.LogEvents = append(logEventsInput.LogEvents, &logEvent)
	logEventsInput.LogGroupName = &logGroupName
	logEventsInput.LogStreamName = &logStreamName

	output, err = cloudwatch.PutLogEvents(&logEventsInput)

	return output, err
}

//IsLogGroupCreated is a function that checks if a particular log group exists or not.
func IsLogGroupCreated(cloudwatch *cloudwatchlogs.CloudWatchLogs, logGroupName string) (bool, error) {
	res := true
	out, err := cloudwatch.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{LogGroupNamePrefix: &logGroupName})
	if len(out.LogGroups) == 0 {
		res = false
	}
	return res, err
}

//IsLogStreamCreated is a function that checks if a particular log group exists or not.
func IsLogStreamCreated(cloudwatch *cloudwatchlogs.CloudWatchLogs, logGroupName string, logStreamName string) (bool, error) {
	res := true
	describeLogStreamsInput := &cloudwatchlogs.DescribeLogStreamsInput{LogGroupName: &logGroupName, LogStreamNamePrefix: &logStreamName}
	out, err := cloudwatch.DescribeLogStreams(describeLogStreamsInput)
	if len(out.LogStreams) == 0 {
		res = false
	}
	return res, err
}
