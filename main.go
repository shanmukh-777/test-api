package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"


	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	_ "github.com/lib/pq"
	"gopkg.in/natefinch/lumberjack.v2"
)
type localcitizen struct{
	NicNum string `json:"nicNum"`
    DateOfBirth string `json:"dateOfBirth"`
    Gender string  `json:"gender"`
    UserType string `json:"userType"`
	LiveImage string `json:"live_image"`
}
type awsConf struct{
	accessKey string
	secretKey string
	kmsKeyId string
	region string
}

	// Define structure for to take input of livelinessSessionresult
	// delete image filed after testing

	// define structure of sessiontable

	// define structure for response of login
type successlogin struct{
		AccessToken string
		RefreshToken string
		AccessTokenExpiry string
	}
	// define structure for customer respone
type verifynicresponse struct{
		CorrelationId string
		NicNum string
		FirstName string
		LastName string
		MaidenName string
		DateOfBirth string
		Photograph string
	}

	
	// define structure for sending final customer data
type customerfinaldata struct{
		NicNum string
		FirstName string
		LastName string
		MaidenName string
		DateOfBirth string
	}

var conf awsConf
var awsConfig *aws.Config
type env struct {
	PostgresHost string   
	PostgresPort  string
	PostgresUser     string
	PostgrePassword  string
	PostgresDbname  string
	MtmlUser string
	MtmlPassword string
	IntPort int
}
var envalues env
func main() {

	SetLogsfilepath()
    //read  environment variable  values for aws config
	conf.accessKey=os.Getenv("AWS_ACCESS_KEY")
    conf.secretKey=os.Getenv("AWS_SECRET_KEY")
	conf.kmsKeyId=os.Getenv("KMS_KEY_ID")
	conf.region=os.Getenv("AWS_REGION")
	envalues.PostgresHost=os.Getenv("POSTGRES_HOST")
	envalues.PostgresPort=  os.Getenv("POSTGRES_PORT")
	envalues.PostgresUser=os.Getenv("POSTGRES_USER")
	envalues.PostgrePassword=  os.Getenv("POSTGRES_PASSWORD")
	envalues.PostgresDbname=os.Getenv("POSTGRES_DBNAME")
	envalues.MtmlUser=os.Getenv("MTML_USER")
	envalues.MtmlPassword=os.Getenv("MTML_PASSWORD")
	port,err :=strconv.Atoi(envalues.PostgresPort)
	if err != nil {
		log.Printf("err:%v",err.Error())
		return
	  }
	envalues.IntPort=port



    // create empty config 
	awsConfig=aws.NewConfig()
	// create a object using static credentials method 
	creds:=credentials.NewStaticCredentials(conf.accessKey,conf.secretKey,"")
	// assign the created object to above created empty config
	awsConfig.WithCredentials(creds)
	awsConfig.WithRegion(conf.region)
    
    // create a engine instance
	router := gin.Default()
    // Define a route that accepts a details of local citizen
    router.POST("/validate-customer", InsertCitizenDetails)
    // Run the server on port 8080
    router.Run(":8080")
}
func SetLogsfilepath() {

	log.SetOutput(&lumberjack.Logger{
		Filename: "./logs/logging.log",
		MaxSize: 64,
		MaxAge: 3,
		MaxBackups: 0,
		Compress: false,
	})
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	log.Println("log file created")
}
func GetAccessToken() (successlogin,error) {
	client := resty.New()

	// Set the base URL of the Swagger-documented API
	apiURL := "https://simapi.icta.mu/icta" // Replace with the actual API URL

	// Specify the endpoint you want to access
	endpoint1 := "/auth/login" // Replace with the actual endpoint

	// Define the request body (assuming it's JSON)

	requestBody1 := map[string]interface{}{
		"password": envalues.MtmlPassword,
		"username": envalues.MtmlUser,
	}

	// Send a POST request with headers and a request body
	response, err := client.R().
		// SetHeaders(headers).
		SetBody(requestBody1).
		Post(apiURL + endpoint1)

	if err != nil {
		// fmt.Println("Error:", err)
		return successlogin{},errors.New("error while sending post request to auth/login api endpoint")
	}

	// Check the HTTP status code
	if response.StatusCode() != 200 {
		stat:=response.StatusCode()
		return successlogin{},errors.New("auth/login api returned a non-200 status code"+strconv.Itoa(stat))
	}
    
	// Process the response body (assuming it's JSON)
	responseBody := response.Body()
	var res map[string]interface{}
	json.Unmarshal(responseBody,&res); 
	accesstoken:= res["accessToken"].(string)
	refreshtoken:=res["refreshToken"].(string)
	accesstokenexpiry:=res["accessTokenExpiryDate"].(string)
	tokenresponse:=successlogin{
		AccessToken: accesstoken,
		RefreshToken: refreshtoken,
		AccessTokenExpiry: accesstokenexpiry,

	}
	return tokenresponse,nil

}
// function to get new acccess token using refresh token endpoint 
func GetNewAccessToken(reftoken string) (successlogin,error) {
	client := resty.New()

	// Set the base URL of the Swagger-documented API
	apiURL := "https://simapi.icta.mu/icta" // Replace with the actual API URL

	// Specify the endpoint you want to access
	endpoint2 := "/auth/refreshToken" // Replace with the actual endpoint

	// Define the request body (assuming it's JSON)
	requestBody2 := map[string]interface{}{
		"refreshToken": reftoken,
	}

	// Send a POST request with headers and a request body
	response, err := client.R().
		// SetHeaders(headers).
		SetBody(requestBody2).
		Put(apiURL + endpoint2)

	if err != nil {
		return successlogin{},errors.New("error while sending post request to auth/refreshToken api endpoint")
	}

	// Check the HTTP status code
	if response.StatusCode() != 200 {
		stat:=response.StatusCode()
		return successlogin{},errors.New("auth/refrshtoken api returned a non-200 status code"+  strconv.Itoa(stat))
	}
    
	// Process the response body (assuming it's JSON)
	responseBody := response.Body()
	var res map[string]interface{}
	json.Unmarshal(responseBody,&res); 
	accesstoken:= res["accessToken"].(string)
	refreshtoken:=res["refreshToken"].(string)
	accesstokenexpiry:=res["accessTokenExpiryDate"].(string)
	tokenresponse:=successlogin{
		AccessToken: accesstoken,
		RefreshToken: refreshtoken,
		AccessTokenExpiry: accesstokenexpiry,

	}
	return tokenresponse,nil

}
func GetCustomerData(token string,data localcitizen) (verifynicresponse,error) {
	client := resty.New()

	// Set the base URL of the Swagger-documented API
	apiURL := "https://simapi.icta.mu/icta" // Replace with the actual API URL

	// Specify the endpoint you want to access
	endpoint3 := "/verifyNIC" // Replace with the actual endpoint

	// Define request headers
	auth:= "Bearer"+"  "+token
	headers := map[string]string{
		 
		 // Adjust the content type as needed
		"Content-Type":  "application/json",
	    "Authorization": auth,  // Replace with your access token  
	    
	}

	requestBody3:= map[string]interface{}{
	    "nicNum": data.NicNum,
        "dateOfBirth": data.DateOfBirth,
        "gender": data.Gender,
        "userType": data.UserType,
	}

	// Send a POST request with headers and a request body
	response, err := client.R().
		SetHeaders(headers).
		SetBody(requestBody3).
		Post(apiURL + endpoint3)

	if err != nil {
		return verifynicresponse{},errors.New("error while sending post request to verifyNic api endpoint")
	}

	// Check the HTTP status code
	if response.StatusCode() != 200 {
		stat:=response.StatusCode() 
		return verifynicresponse{},errors.New("auth/login api returned a non-200 status code"+strconv.Itoa(stat))

	}

	// Process the response body (assuming it's JSON)
	responseBody := response.Body()
	
	// accesstoken:=access.
	fmt.Println("Response:", string(responseBody))
	// var res map[string]interface{}
	var customerdata verifynicresponse
	json.Unmarshal(responseBody,&customerdata); 
	return customerdata,nil

}
func InsertCitizenDetails(c *gin.Context){
	var details localcitizen
	if err := c.ShouldBindJSON(&details); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
	// creating a connection string for postgres
    tokendata,err:=GetAccessToken()
	if err != nil {
		c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
		log.Printf("err:%v",err.Error())
		return
	}
	log.Printf("token recieved succesful")

	currentTime := time.Now()
	// checking the expire of access token
	if currentTime.String() > tokendata.AccessTokenExpiry {
		// calling refresh token endpoint of mtml using function
		newtokendata,err:=GetNewAccessToken(tokendata.RefreshToken)
		if err != nil {
			c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
			log.Printf("err:%v",err.Error())
			return
		}
		tokendata.AccessToken=newtokendata.AccessToken
		tokendata.RefreshToken=newtokendata.RefreshToken
		tokendata.AccessTokenExpiry=newtokendata.AccessTokenExpiry
		log.Printf("token refresh succesful")

	}
	// data of a customer from mtml endpoint 
	customer,err:=GetCustomerData(tokendata.AccessToken,details)
	if err != nil {
		c.JSON(http.StatusBadRequest,gin.H{"error":err.Error()})
		log.Printf("err:%v",err.Error())
		return
	}
	log.Printf("customer data recieved succesful")
	liveimage:=details.LiveImage
	refimage:=customer.Photograph
	myresSession := session.Must(session.NewSession())
	livres := rekognition.New(myresSession,awsConfig)
	sourcebytes,err := base64.StdEncoding.DecodeString(liveimage)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot decode b64"})
		log.Printf("err:%v",err.Error())
		return
	}
	sourceImage := rekognition.Image{
		Bytes: sourcebytes, // Replace with actual image data
	}
	// facesrequest.SetSourceImage(&sourceImage)
	//reference image
	refbytes,err := base64.StdEncoding.DecodeString(refimage)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot decode b64"})
		log.Printf("err:%v",err.Error())
		return
	}
	refImage := rekognition.Image{
		Bytes: refbytes,
	
	}
	// facesrequest.SetTargetImage(&refImage)
    facesRequest := rekognition.CompareFacesInput {
		SourceImage: &sourceImage,
		TargetImage: &refImage,
		SimilarityThreshold: aws.Float64(70),	
			}	



	resu,err:=livres.CompareFaces(&facesRequest)
	if err !=nil{
        c.JSON(http.StatusBadRequest, gin.H{"error": "error while sending api request to aws"})
		log.Printf("err:%v",err.Error())
		return
	}
	// similarity and confidence of compare faces functionality of aws rekognition are stored in faceDetails.Similarity and faceDetails.Face.Confidence
	faceDetails:= resu.FaceMatches[0]
	sendingcustomerdata:=customerfinaldata{
		NicNum: customer.NicNum,
		FirstName: customer.FirstName,
		LastName: customer.LastName,
		MaidenName: customer.MaidenName,
		DateOfBirth: customer.DateOfBirth,
	}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
    "password=%s dbname=%s sslmode=disable",
    envalues.PostgresHost, envalues.IntPort, envalues.PostgresUser,envalues.PostgrePassword, envalues.PostgresDbname)
	// opening a connection to database
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
	  c.JSON(http.StatusBadRequest,gin.H{"error":"unable to open connection to postgress","errormessage":err.Error()})
	  log.Printf("err:%v",err.Error())
	  return
	}
	defer db.Close()
    // validate the connection
	err = db.Ping()
	if err != nil {
		c.JSON(http.StatusBadRequest,gin.H{"error":"error while validating connection to postgress","errormessage":err.Error()})
		log.Printf("err:%v",err.Error())
		return
	}
	ctime:=time.Now()
    sqlStatement := `
    INSERT INTO localcitizens (nicnum, dateofbirth, gender, usertype,image,time)
    VALUES ($1,$2,$3,$4,$5,$6)`
    _, err = db.Exec(sqlStatement,details.NicNum,details.DateOfBirth,details.Gender,details.UserType,liveimage,ctime)
    if err != nil {
		c.JSON(http.StatusBadRequest,gin.H{"error":"unable to write data into postgress","errormessage":err.Error()})
		log.Printf("err:%v",err.Error())
		return
    }
	log.Printf("data of citizen saved successfully")
	c.JSON(http.StatusOK,gin.H{
      "comparefaces result": gin.H{
		"similarity":faceDetails.Similarity,
		"confidence":faceDetails.Face.Confidence,
		} ,
       "customer details": sendingcustomerdata})
	log.Printf("liveliness,compareface and customer data successfully sent")

}