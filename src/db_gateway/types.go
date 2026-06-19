package dbgateway

// BreadcrumbFile represents a folder information. This is used
// with the recursive CTE call for the breadcrumbs navigation.
type BreadcrumbFile struct {
	FileId   string `json:"fileId"`
	ParentId string `json:"parentId"`
	Name     string `json:"fileName"`
}
