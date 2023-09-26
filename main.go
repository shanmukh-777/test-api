package main

import (
	"bytes"
	"encoding/base64"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
)
type request struct {
	Mimetype    string `json:"mime_type"`
	Base64Image string `json:"base64_image"`
}


type facescompare struct{
	Liveface string `json:"live_face"`
	Referface string `json:"ref_face"`
	similaritythres float64 `json:"sim_thres"`
}

type awsConf struct{
accessKey string
secretKey string
kmsKeyId string
region string
}
type liveliness struct {
	Uniquetoken string `json:"unique_token"`
}

var conf awsConf
var awsConfig *aws.Config
func main() {

	SetLogsfilepath()

	conf.accessKey=os.Getenv("AWS_ACCESS_KEY")
    conf.secretKey=os.Getenv("AWS_SECRET_KEY")
	conf.kmsKeyId=os.Getenv("KMS_KEY_ID")
	conf.region=os.Getenv("AWS_REGION")
   
	awsConfig=aws.NewConfig()
	creds:=credentials.NewStaticCredentials(conf.accessKey,conf.secretKey,"")
	awsConfig.WithCredentials(creds)
	awsConfig.WithRegion(conf.region)

    // Define a route that accepts a Base64 encoded image
    router := gin.Default()
    router.POST("/insert-image", InsertBase64Image)
	router.POST("/compare-face", CompareFaces)
    router.POST("/create-liveliness-session", CreateLive)
	// router.GET("/create-kmskey",CreateK)
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


func InsertBase64Image(c *gin.Context) {
    // Get the Base64 encoded image data from the request body
	var req request


    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }


	unbased, err := base64.StdEncoding.DecodeString(req.Base64Image)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot decode b64"})
		return
	}
    if req.Mimetype == "image/png" {
		r := bytes.NewReader(unbased)
		im, err := png.Decode(r)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad png"})
			log.Printf("err:%v",err.Error())
			return
		}

	
		f, err := os.OpenFile("./images/example.png", os.O_WRONLY|os.O_CREATE, 0777)
    	if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot open file for png"})
			log.Printf("err:%v",err.Error())

        	return
   		 }
		err =png.Encode(f,im)
		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write png image"})
			log.Printf("err:%v",err.Error())

			return
		}
	}else if req.Mimetype == "image/jpeg"{
		q :=bytes.NewReader(unbased)
		imm ,err := jpeg.Decode(q)
		if err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad png"})
			log.Printf("err:%v",err.Error())

			return
		}

		fi,err :=os.OpenFile("./images/exp"+time.Now().String()+".jpeg",os.O_WRONLY|os.O_CREATE, 0777)
    	if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot open file for jpeg"})
			log.Printf("err:%v",err.Error())

        	return
   		 }	
		err = jpeg.Encode(fi,imm,nil)
		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write jpeg image"})
			log.Printf("err:%v",err.Error())

			return
		}		


	}else{
		c.JSON(http.StatusBadRequest,gin.H{
			"error":"unknown mime type",
		})
		log.Print("err: unknown mime type")
		return 
	}

    // Respond with a success message
    c.JSON(http.StatusOK, gin.H{"message": "Image saved successfully"})
	log.Printf("Image saved successfully")

}
func CompareFaces(c *gin.Context){
	var faces facescompare
	
    if err := c.ShouldBindJSON(&faces); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    mySession := session.Must(session.NewSession())
	
	svc := rekognition.New(mySession, awsConfig)

	// similarityThreshold := &faces.similaritythres
	// facesrequest.SetSimilarityThreshold(*similarityThreshold)
	//source image
	sourcebytes,err := base64.StdEncoding.DecodeString(faces.Liveface)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot decode b64"})
		return
	}
	sourceImage := rekognition.Image{
		Bytes: sourcebytes, // Replace with actual image data
	}
	// facesrequest.SetSourceImage(&sourceImage)
	//reference image
	refbytes,err := base64.StdEncoding.DecodeString(faces.Referface)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot decode b64"})
		return
	}
	refImage := rekognition.Image{
		Bytes: refbytes,
	
	}
	// facesrequest.SetTargetImage(&refImage)
    facesRequest := rekognition.CompareFacesInput {
		SourceImage: &sourceImage,
		TargetImage: &refImage,
		SimilarityThreshold: &faces.similaritythres,	
			}	



	res,err:=svc.CompareFaces(&facesRequest)
	if err !=nil{
        c.JSON(http.StatusBadRequest, gin.H{"error": "error while sending api request to aws"})
		return
	}
	faceDetails:= res.FaceMatches[0]
	sim:=faceDetails.Similarity
	conf:=faceDetails.Face.Confidence
    c.JSON(http.StatusOK, gin.H{"similarity": sim,
          "confidence": conf })
	
}	
// func CreateK(c *gin.Context){
// 	mylivSession := session.Must(session.NewSession())
//     kmsge:=kms.New(mylivSession)
	
// 	input:=kms.CreateKeyInput{
// 		Description: aws.String("my kms key") ,
// 		KeyUsage: aws.String("ENCRYPT_DECRYPT"),

// 	}
// 	resut,err:=kmsge.CreateKey(&input)
// 	if err !=nil{
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err})
// 		return	
// 	}
// 	keyi:=resut.KeyMetadata.KeyId
// 	c.JSON(http.StatusOK,gin.H{"keyid": keyi})

// }


func CreateLive(c *gin.Context){

	var livetoken liveliness
	mylivSession := session.Must(session.NewSession())
	
	liv := rekognition.New(mylivSession,awsConfig)
	
	
	limit:=int64(1)
	outputconfi:=rekognition.CreateFaceLivenessSessionRequestSettings{
		AuditImagesLimit: &limit ,
	}
	livenessfaces:=rekognition.CreateFaceLivenessSessionInput{
		ClientRequestToken: &livetoken.Uniquetoken,
		KmsKeyId:           &conf.kmsKeyId,
		Settings:           &outputconfi,
	}
	responseB,err:=liv.CreateFaceLivenessSession( &livenessfaces)
	if err != nil{
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sessionID": responseB})

}


    

	

