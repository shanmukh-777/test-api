package main

import (
	"bytes"
	"image/png"
	"image/jpeg"
	"os"
	"time"
    "encoding/base64"
    "github.com/gin-gonic/gin"
    "net/http"
	"log"
	"gopkg.in/natefinch/lumberjack.v2"

)
type request struct {
	Mimetype    string `json:"mime-type"`
	Base64Image string `json:"base64_image"`
}



func main() {

	SetLogsfilepath()
	
	// Create a new Gin router
    router := gin.Default()

    // Define a route that accepts a Base64 encoded image
    router.POST("/insert-image", InsertBase64Image)

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
