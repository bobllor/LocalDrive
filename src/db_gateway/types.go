package dbgateway

// FileFolderInfo represents a folder information. This is used
// with the recursive CTE call for the breadcrumbs navigation.
type FileFolderInfo struct {
	FileId   string `json:"fileId"`
	ParentId string `json:"parentId"`
	Name     string `json:"fileName"`
}
