package api

// RequestUserLoginInfo contains the login information of the user.
type RequestUserLoginInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RequestUserRegisterInfo contains the login information of the user
// for registering an account.
type RequestUserRegisterInfo struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

// RequestFileUploadInfo is used to hold metadata of the file being uploaded.
type RequestFileUploadInfo struct {
	FileName      string `json:"fileName"`
	FileSize      int    `json:"fileSize"`
	FileExtension string `json:"fileExtension"`
	FileParentId  string `json:"fileParentId"`
	TotalChunks   int    `json:"totalChunks"`
}

// RequestAddFolderInfo contains the request body for adding a folder to the
// database and organization of files for the front end.
type RequestAddFolderInfo struct {
	Name     string `json:"fileName"`
	ParentId string `json:"parentId"`
}

// RequestRenameFile is the request body to rename a file.
type RequestRenameFile struct {
	FileId      string `json:"fileId"`
	NewFileName string `json:"newFileName"`
}

// RequestDeleteFile is the request body for deleting a file.
type RequestDeleteFile struct {
	FileIds []string `json:"fileIds"`
}
