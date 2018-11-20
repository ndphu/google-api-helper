package google_api_helper

import (
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"io/ioutil"
)

type Quota struct {
	Limit   int64  `json:"limit"`
	Usage   int64  `json:"usage"`
	Percent string `json:"percent"`
}

type File struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Size int64 `json:"size"`
}

func InitGoogleDriveServiceFromFile(path string) (*drive.Service) {
	b, err := ioutil.ReadFile(path)
	FailOnError("Unable to read client secret file", err)
	return InitGoogleDriveService(b, drive.DriveScope)
}

func InitGoogleDriveService(token []byte, scope string) (*drive.Service) {
	config, err := google.JWTConfigFromJSON(token, scope)
	FailOnError("Unable to parse client secret file to config", err)
	client := config.Client(oauth2.NoContext)
	srv, err := drive.New(client)
	FailOnError("Unable to retrieve Drive client", err)
	return srv
}

func GetQuotaUsage(srv *drive.Service) *Quota {
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

func ListFiles(srv *drive.Service, page int, size int64) []File {
	if page < 1 {
		FailOnError(fmt.Sprintf("Invalid page %d", page), nil)
	}

	if page == 1 {
		return retrieveFiles(srv, "", size)
	} else {
		pageToken := getPageToken(srv, page, size)
		files := retrieveFiles(srv, pageToken, size)
		return files
	}

}

func getPageToken(srv *drive.Service, page int, size int64) string {
	call := srv.Files.List().PageSize(size)
	pageToken := ""
	for cp := 1; cp < page; cp ++ {
		token, err := call.Fields("nextPageToken").PageToken(pageToken).Do()
		FailOnError("Fail to get next page", err)
		pageToken = token.NextPageToken
	}
	return pageToken
}

func retrieveFiles(srv *drive.Service, pageToken string, size int64) []File {
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

func DeleteAllFiles(srv *drive.Service) {
	r, err := srv.Files.List().Fields("files(id, name)").Do()

	if err != nil {
		FailOnError("Fail to list file", err)
	}
	for _, file := range r.Files {
		srv.Files.Delete(file.Id).Do()
	}
}
