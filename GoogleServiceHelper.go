package google_api_helper

import (
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/drive/v3"
)

type Quota struct {
	Limit   int64  `json:"limit"`
	Usage   int64  `json:"usage"`
	Percent string `json:"percent"`
}

type File struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type DriveService struct {
	Service *drive.Service
	Config  *jwt.Config
}

func GetDriveService(token []byte) *DriveService {
	config, err := google.JWTConfigFromJSON(token, drive.DriveScope)
	FailOnError("Unable to parse client secret file to config", err)
	client := config.Client(oauth2.NoContext)
	srv, err := drive.New(client)
	FailOnError("Unable to retrieve Drive client", err)
	return &DriveService{
		Service: srv,
		Config:  config,
	}
}

func (d *DriveService) GetQuotaUsage() *Quota {
	srv := d.Service
	about, err := srv.About.Get().Fields("user,storageQuota").Do()
	if err != nil {
		FailOnError("Unable to retrieve About client", err)
	}
	return &Quota{
		Limit:   about.StorageQuota.Limit,
		Usage:   about.StorageQuota.Usage,
		Percent: fmt.Sprintf("%.3f", float64(about.StorageQuota.Usage)*100/float64(about.StorageQuota.Limit)),
	}
}

func (d *DriveService) ListFiles(page int, size int64) []File {
	if page < 1 {
		FailOnError(fmt.Sprintf("Invalid page %d", page), nil)
	}

	if page == 1 {
		return d.retrieveFiles("", size)
	} else {
		pageToken := d.getPageToken( page, size)
		files := d.retrieveFiles(pageToken, size)
		return files
	}

}

func (d *DriveService) getPageToken(page int, size int64) string {
	srv := d.Service
	call := srv.Files.List().PageSize(size)
	pageToken := ""
	for cp := 1; cp < page; cp ++ {
		token, err := call.Fields("nextPageToken").PageToken(pageToken).Do()
		FailOnError("Fail to get next page", err)
		pageToken = token.NextPageToken
	}
	return pageToken
}

func (d *DriveService) retrieveFiles(pageToken string, size int64) []File {
	srv := d.Service
	call := srv.Files.List().PageSize(size)
	r, err := call.PageToken(pageToken).Fields("files(id, name, size)").Do()
	FailOnError("Fail to list file", err)
	files := make([]File, len(r.Files))
	for i, file := range r.Files {
		files[i] = File{
			Id:   file.Id,
			Name: file.Name,
			Size: size,
		}
	}
	return files
}

func (d *DriveService) DeleteAllFiles() {
	srv := d.Service
	r, err := srv.Files.List().Fields("files(id, name)").Do()

	if err != nil {
		FailOnError("Fail to list file", err)
	}
	for _, file := range r.Files {
		srv.Files.Delete(file.Id).Do()
	}
}

func (d *DriveService) GetDownloadLink(fileId string) string {
	accessToken, err:=d.Config.TokenSource(oauth2.NoContext).Token()
	FailOnError("Fail to get access token", err)
	return fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media&prettyPrint=false&access_token=%s",
		fileId, accessToken.AccessToken)
}
