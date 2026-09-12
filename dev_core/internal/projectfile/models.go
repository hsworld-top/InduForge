package projectfile

import "time"

// File 是工程对象库对外公开的文件元数据；object_key 永远不会出现在此结构中。
type File struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	UpdatedAt   time.Time `json:"updatedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	Entry       string    `json:"entry"`
}

type ListFilter struct {
	Path    string
	Keyword string
	Page    int
	Limit   int
}
