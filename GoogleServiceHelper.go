package google_api_helper

import (
	"errors"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/drive/v3"
	"io"
	"log"
	"os"
	"strings"
)

type Quota struct {
	Limit   int64  `json:"limit"`
	Usage   int64  `json:"usage"`
	Percent string `json:"percent"`
}

type File struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
}

type DriveService struct {
	Service *drive.Service
	Config  *jwt.Config
}

type DownloadDetails struct {
	Link string `json:"link"`
	Token string `json:"token"`
	UserAgent string `json:"userAgent"`
	XApiClient string `json:"xApiClient"`
}


var RedirectAttemptedError = errors.New("redirect")

func GetDriveService(token []byte) (*DriveService, error) {
	config, err := google.JWTConfigFromJSON(token, drive.DriveScope)
	if err != nil {
		return nil, err
	}
	client := config.Client(oauth2.NoContext)
	srv, err := drive.New(client)
	if err != nil {
		return nil, err
	}
	return &DriveService{
		Service: srv,
		Config:  config,
	}, nil
}

func (d *DriveService) GetQuotaUsage() (*Quota, error) {
	srv := d.Service
	about, err := srv.About.Get().Fields("user,storageQuota").Do()
	if err != nil {
		return nil, err
	}
	return &Quota{
		Limit:   about.StorageQuota.Limit,
		Usage:   about.StorageQuota.Usage,
		Percent: fmt.Sprintf("%.3f", float64(about.StorageQuota.Usage)*100/float64(about.StorageQuota.Limit)),
	}, nil
}

func (d *DriveService) ListFiles(page int, size int64) ([]File, error) {
	if page == 1 {
		return d.retrieveFiles("", size)
	} else {
		pageToken, err := d.getPageToken(page, size)
		if err != nil {
			return nil, err
		}
		return d.retrieveFiles(pageToken, size)
	}

}

func (d *DriveService) getPageToken(page int, size int64) (string, error) {
	srv := d.Service
	call := srv.Files.List().PageSize(size)
	pageToken := ""
	for cp := 1; cp < page; cp ++ {
		token, err := call.Fields("nextPageToken").PageToken(pageToken).Do()
		if err != nil {
			return "", err
		}
		pageToken = token.NextPageToken
	}
	return pageToken, nil
}

func (d *DriveService) retrieveFiles(pageToken string, size int64) ([]File, error) {
	srv := d.Service
	call := srv.Files.List().PageSize(size)
	r, err := call.PageToken(pageToken).Fields("files(id, name, size, mimeType)").Do()
	//FailOnError("Fail to list file", err)
	if err != nil {
		return nil, err
	}
	files := make([]File, len(r.Files))
	for i, file := range r.Files {
		files[i] = File{
			Id:       file.Id,
			Name:     file.Name,
			Size:     file.Size,
			MimeType: file.MimeType,
		}
	}
	return files, nil
}

func (d *DriveService) DeleteAllFiles() (error) {
	srv := d.Service
	r, err := srv.Files.List().Fields("files(id, name)").Do()

	if err != nil {
		return err
	}

	for _, file := range r.Files {
		err := srv.Files.Delete(file.Id).Do()
		if err != nil {
			return err
		}
	}
	return nil
}

//<<<<<<< Updated upstream
//func (d *DriveService) GetDownloadLink(fileId string) (*drive.File, *DownloadDetails, error) {
//	file, err := d.Service.Files.Get(fileId).Fields("id, name, size, mimeType, webContentLink, webViewLink, shared").Do()
//
//	if err != nil {
//		return nil, nil, err
//=======
//func (d *DriveService) GetDownloadLink(fileId string) (*drive.File, string, error) {
//	file, err := d.Service.Files.Get(fileId).Fields("id, name, size, mimeType, webContentLink").Do()
//	if err != nil {
//		return nil, "", err
//	}
//	log.Println("webContentLink",file.WebContentLink)
//
//	log.Println("https://drive.google.com/uc?export=download&amp;confirm=MEBm&amp;id="+file.Id)
//	accessToken, err := d.Config.TokenSource(oauth2.NoContext).Token()
//	if err != nil {
//		return nil, "", err
//	}
//	log.Println("accessToken.AccessToken", accessToken.AccessToken)
//	fileUrl := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media&prettyPrint=false&access_token=%s",
//		fileId, accessToken.AccessToken)
//
//	log.Println("fileUrl", fileUrl)
//
//	client := &http.Client{
//		CheckRedirect: func(req *http.Request, via []*http.Request) error {
//			return RedirectAttemptedError
//		},
//>>>>>>> Stashed changes
//	}
//	res, err := d.Service.Files.Get(fileId).Download()
//
//	if err == nil {
//		log.Println(res.Request.URL.String(), res.Request.UserAgent())
//		log.Println(res.Request.Header)
//
//		defer res.Body.Close()
//	}
//	return file, &DownloadDetails{
//		Link: res.Request.URL.String(),
//		UserAgent: res.Request.UserAgent(),
//		Token: strings.TrimPrefix(res.Request.Header.Get("Authorization"), "Bearer "),
//		XApiClient: res.Request.Header.Get("X-Goog-Api-Client"),
//	}, nil
//}
func (d *DriveService) GetDownloadLink(fileId string) (*drive.File, *DownloadDetails, error) {
	file, err := d.Service.Files.Get(fileId).Fields("id, name, size, mimeType, webContentLink, webViewLink, shared").Do()

	if err != nil {
		return nil, nil, err
	}
	res, err := d.Service.Files.Get(fileId).Download()

	if err == nil {
		log.Println(res.Request.URL.String(), res.Request.UserAgent())
		log.Println(res.Request.Header)

		defer res.Body.Close()
	}
	return file, &DownloadDetails{
		Link: res.Request.URL.String(),
		UserAgent: res.Request.UserAgent(),
		Token: strings.TrimPrefix(res.Request.Header.Get("Authorization"), "Bearer "),
		XApiClient: res.Request.Header.Get("X-Goog-Api-Client"),
	}, nil
}
func (d *DriveService) UploadFile(name string, description string, mimeType string, localPath string) (*drive.File, error) {
	localFile, err := os.Open(localPath)
	if err != nil {
		log.Fatalf("error opening %q: %v", name, err)
	}
	defer localFile.Close()
	f := &drive.File{Name: name, Description: description, MimeType: mimeType}
	return d.Service.Files.Create(f).Media(localFile).Do()
}

func (d *DriveService) UploadFileFromStream(name string, description string, mimeType string, is io.Reader) (*drive.File, error) {
	f := &drive.File{Name: name, Description: description, MimeType: mimeType}
	return d.Service.Files.Create(f).Media(is).Do()
}

func (d *DriveService) GetSharableLink(fileId string) (*drive.File, string, error) {
	perm := drive.Permission{
		Type: "anyone",
		Role: "reader",
	}
	_, err := d.Service.Permissions.Create(fileId, &perm).Do()
	if err != nil {
		return nil, "", err
	}

	file, err := d.Service.Files.Get(fileId).Fields("id, name, size, mimeType").Do()
	if err != nil {
		return nil, "", err
	}
	fileUrl := fmt.Sprintf("https://drive.google.com/file/d/%s/view", fileId)

	return file, fileUrl, nil
}

func (d *DriveService) GetAccessToken() (string, error) {
	token, err := d.Config.TokenSource(oauth2.NoContext).Token()
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}
