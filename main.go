package main

import (
	"archive/zip"
	"errors"
	"fmt"
	chatgpt "go-gpt/chat-gpt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const (
	TEMP_DIR         string = "tmp/"
	TEMP_DIR_RESULTS string = TEMP_DIR + "results/"
	REPORT_FILE_NAME string = "report.md"
)

var chatGpt *chatgpt.ChatGpt
var wg sync.WaitGroup

func main() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	apiKey := os.Getenv("API_KEY")
	organization := os.Getenv("API_ORG")
	chatGpt = chatgpt.NewClient(organization, apiKey)

	router := gin.Default()
	router.POST("/chat-gpt/analyze", analyzeFile)

	router.Run("localhost:8080")
}

func analyzeFile(c *gin.Context) {
	// extract file from payload
	file, err := c.FormFile("file")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("File name received:", file.Filename)

	// save the file temporary in order to handle it later on
	err = c.SaveUploadedFile(file, TEMP_DIR+file.Filename)
	if err != nil {
		log.Fatal(err)
	}

	// unzip file
	archive, err := zip.OpenReader(TEMP_DIR + file.Filename)
	if err != nil {
		log.Fatal(err)
	}
	defer archive.Close()

	programmingFiles := make(map[string]string)

	for _, f := range archive.File {
		ext := filepath.Ext(f.Name)
		if ext == ".go" {
			// Open the current file
			v, err := f.Open()
			if err != nil {
				log.Fatal(err)
			}
			defer v.Close()

			// Read the contents of the file
			b, err := io.ReadAll(v)
			if err != nil {
				log.Fatal(err)
			}

			programmingFiles[f.Name] = string(b)
		}
	}

	createFeedbackReport(programmingFiles)

	content, err := os.ReadFile(TEMP_DIR_RESULTS + REPORT_FILE_NAME)
	if err != nil {
		log.Fatal(err)
	}
	c.Header("Content-Disposition", "attachment; filename="+REPORT_FILE_NAME)
	c.Header("Content-Type", "application/text/plain")
	c.Header("Accept-Length", fmt.Sprintf("%d", len(content)))
	c.Writer.Write(content)
	c.JSON(http.StatusOK, nil)

	defer os.RemoveAll("./tmp")
}

func requestFeedback(fileName string, fileContent string, c chan string) {
	defer wg.Done()

	log.Print("Requesting feedback for file", fileName)

	msg := "Act as a developer, and without modifying the original file, add some comments in the code with suggestions and improvements. Also give me a general summary of the code quality first" + fileContent

	response, err := chatGpt.ChatCompletion(msg)
	if err != nil {
		log.Fatal(err)
		return
	}

	result := "## Feedback related to file" + "`" + fileName + "`" + "\n\n" + response + "\n\n" + "----------------------------------" + "\n\n"
	c <- result
}

func createFeedbackReport(programmingFiles map[string]string) {
	feedbacks := ""
	wg.Add(len(programmingFiles))
	ch := make(chan string, len(programmingFiles))
	for key, value := range programmingFiles {
		go requestFeedback(key, value, ch)
	}
	go func() {
		for c := range ch {
			feedbacks += c
		}
	}()
	wg.Wait()
	createFile(feedbacks)
}

func createFile(text string) {
	// create directory to extract files to
	if _, err := os.Stat(TEMP_DIR_RESULTS); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(TEMP_DIR_RESULTS, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
	// Write files to directory
	newFile, err := os.Create(TEMP_DIR_RESULTS + REPORT_FILE_NAME)
	if err != nil {
		log.Fatal(err)
	}
	defer newFile.Close()
	newFile.WriteString(text)
}
