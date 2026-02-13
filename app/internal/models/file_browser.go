package models

type FileEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
	ModifiedAt int64  `json:"modifiedAt"`
}

type ListFilesResult struct {
	Path    string      `json:"path"`
	Entries []FileEntry `json:"entries"`
}
