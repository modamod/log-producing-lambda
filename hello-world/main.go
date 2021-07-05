package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

var (
	// DefaultHTTPGetAddress Default Address
	DefaultHTTPGetAddress = "https://checkip.amazonaws.com"

	// ErrNoIP No IP found in response
	ErrNoIP = errors.New("No IP in HTTP response")

	// ErrNon200Response non 200 status code in response
	ErrNon200Response = errors.New("Non 200 Response found")
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	resp, err := http.Get(DefaultHTTPGetAddress)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if resp.StatusCode != 200 {
		return events.APIGatewayProxyResponse{}, ErrNon200Response
	}

	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if len(ip) == 0 {
		return events.APIGatewayProxyResponse{}, ErrNoIP
	}

	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf("Hello, %v", string(ip)),
		StatusCode: 200,
	}, nil
}

func main() {
	// WriteToDynamodb()
	// tpl, _ := gonja.FromString("{{ range(0, 99999) | random }}")
	// out, _ := tpl.Execute(nil)
	// fmt.Println(out)
	region := "us-east-1"
	profile := "default"

	config := aws.Config{
		Region: aws.String(region),
	}
	sessionOptions := session.Options{Config: config, Profile: profile}
	session, err := session.NewSessionWithOptions(sessionOptions)

	if err != nil {
		log.Fatalln("Failed to create session using Profile: " + profile + err.Error())
	}
	cloudwatch := cloudwatchlogs.New(session)
	logGrouptInput := cloudwatchlogs.CreateLogGroupInput{LogGroupName: aws.String("/lambda/test/log-group")}
	logStreamInput := cloudwatchlogs.CreateLogStreamInput{LogGroupName: aws.String("/lambda/test/log-group"), LogStreamName: aws.String("/lambda/test/log-group-stream")}

	context, err := ContextFromFile("templates/log.parameters")
	if err != nil {
		log.Fatalln("Failed to parse parameter file." + err.Error())
	}

	msg, err := GenerateLogs("log.template", context)
	if err != nil {
		log.Fatalln("Failed to generate log from template file." + err.Error())
	}

	cloudwatch.CreateLogStream(&logStreamInput)
	logGroupCreated, err := IsLogGroupCreated(cloudwatch, *logGrouptInput.LogGroupName)
	if err != nil {
		log.Fatalln("Failed to get log groups with LogGroupNamePrefix: " + *logGrouptInput.LogGroupName)
	}
	if !logGroupCreated {
		cloudwatch.CreateLogGroup(&logGrouptInput)

	}
	logStreamCreated, err := IsLogStreamCreated(cloudwatch, *logGrouptInput.LogGroupName, *logStreamInput.LogStreamName)
	if err != nil {
		log.Fatalln("Failed to get log Streams with LogStreamNamePrefix: " + *logStreamInput.LogStreamName)
	}
	if !logStreamCreated {
		cloudwatch.CreateLogStream(&logStreamInput)

	}
	fmt.Println(logGroupCreated)
	out, err := PutLogEvents(cloudwatch, *logGrouptInput.LogGroupName, *logStreamInput.LogStreamName, &msg)
	if err != nil {
		log.Fatalln("Failed to put log event")
	}
	fmt.Println(out, err)
	lambda.Start(handler)
}
