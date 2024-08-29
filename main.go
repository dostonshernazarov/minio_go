package main

import (
	"context"
	"fmt"
	_ "lessons/minio/docs"
	"mime/multipart"
	"path/filepath"
	"time"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type File struct {
	File multipart.FileHeader `form:"file" binding:"required"`
}

func main() {

	router := gin.Default()

	router.POST("/media", Media)

	url := ginSwagger.URL("swagger/doc.json")
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))

	router.Run(":50040")
}

// uploadFile
// @Summary uploadFile
// @Description Upload a media file
// @Tags media
// @Accept multipart/form-data
// @Param file formData file true "UploadMediaForm"
// @Success 201 {object} string
// @Failure 400 {object} error
// @Failure 500 {object} error
// @Router /media [post]
func Media(c *gin.Context) {
	var file File
	err := c.ShouldBind(&file)
	if err != nil {
		c.AbortWithStatusJSON(400, err)
		return
	}

	fileUrl := filepath.Join("./media", file.File.Filename)

	err = c.SaveUploadedFile(&file.File, fileUrl)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	fileExt := filepath.Ext(file.File.Filename)

	newFile := uuid.NewString() + fileExt

	minioClient, err := minio.New("3.70.205.207:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("test", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
				{
						"Effect": "Allow",
						"Principal": {
								"AWS": ["*"]
						},
						"Action": ["s3:GetObject"],
						"Resource": ["arn:aws:s3:::%s/*"]
				}
		]
}`, "photos")

	// err = minioClient.MakeBucket(context.Background(), "photos", minio.MakeBucketOptions{})
	// if err != nil {
	// 	c.AbortWithError(500, err)
	// 	return
	// }

	info, err := minioClient.FPutObject(context.Background(), "photos", newFile, fileUrl, minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	err = minioClient.SetBucketPolicy(context.Background(), "photos", policy)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	println("\n Info Bucket:", info.Bucket)

	objUrl, err := minioClient.PresignedGetObject(context.Background(), "photos", newFile, time.Hour*24, nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	madeUrl := fmt.Sprintf("http://3.70.205.207:9000/photos/%s", newFile)

	c.JSON(201, gin.H{
		"url":      objUrl.String(),
		"made_url": madeUrl,
	})

}
