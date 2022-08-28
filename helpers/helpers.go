package helpers

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Vasanth-Korada/sitemap-crawler/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xuri/excelize/v2"
)

/*Function to return user agents (just to specify that our program is from a browser)*/
func GetUserAgents() []string {
	return []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:56.0) Gecko/20100101 Firefox/56.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
	}
}

/*Function to check if the given url is a sitemap*/
func IsSitemap(urls []string) ([]string, []string) {
	siteMapFiles := []string{}
	pages := []string{}
	for _, link := range urls {
		foundSitemap := strings.Contains(link, "xml")
		if foundSitemap {
			fmt.Println("Found Sitemap", link)
			siteMapFiles = append(siteMapFiles, link)
		} else {
			pages = append(pages, link)
		}
	}
	return siteMapFiles, pages
}

// Functions which generates an excel file
func GenerateExcelFile(scrapedData []models.SEOData) {
	xlFile := excelize.NewFile()
	for i, seoData := range scrapedData {
		row := i + 1
		xlFile.SetCellValue("Sheet1", "A"+strconv.Itoa(row), seoData.URL)
		row++
		xlFile.SetCellValue("Sheet1", "B"+strconv.Itoa(row), seoData.Title)
		row++
		xlFile.SetCellValue("Sheet1", "C"+strconv.Itoa(row), seoData.H1)
		row++
		xlFile.SetCellValue("Sheet1", "D"+strconv.Itoa(row), seoData.StatusCode)

		xlFile.SetColWidth("Sheet1", "A", "D", 50)
	}
	if err := xlFile.SaveAs("scraped_data.xlsx"); err != nil {
		log.Fatalf("Error: %s", err)
		return
	}
	fmt.Println("-------------------------------------")
	fmt.Println("Scraped Data Downloaded Successfully!")
	UploadFileToS3("./scraped_data.xlsx")
}

// Functions to load AWS Config
func loadAWSConfig() *s3.Client {

	creds := credentials.NewStaticCredentialsProvider(os.Getenv("ACCESS_KEY_ID"), os.Getenv("SECRET_ACCESS_KEY"), "")

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(creds), config.WithRegion(os.Getenv("S3_REGION")))

	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Uploading to S3....")
	}

	return s3.NewFromConfig(cfg)

}

// Functions to upload file to AWS S3
func UploadFileToS3(imageFile string) error {

	// Open the file from the file path
	upFile, err := os.Open(imageFile)
	if err != nil {
		return fmt.Errorf("could not open local filepath [%v]: %+v", imageFile, err)
	}
	defer upFile.Close()

	uploader := manager.NewUploader(loadAWSConfig())
	uploadResult, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String("scrapeddataaws"),
		Key:         aws.String("data/scraped_data.xlsx"),
		Body:        upFile,
		ContentType: aws.String("vnd.ms-excel"),
	})

	if err != nil {
		fmt.Printf("Error: %v  \n", err)
		return err
	}
	fmt.Printf("Scraped Data File has been uploaded ->> %v \t \n", imageFile)
	fmt.Println("Download URL: " + uploadResult.Location)
	fmt.Println("-------------------------------------")
	return nil
}
