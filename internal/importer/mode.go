package importer

type ImportMode string

const (
	ModeReplace ImportMode = "replace"
	ModeMerge   ImportMode = "merge"
)

type Result struct {
	Servers   int `json:"servers"`
	Tables    int `json:"tables"`
	Functions int `json:"functions"`
}

func CountResult(doc DeclarativeDoc) Result {
	return Result{
		Servers:   len(doc.Servers),
		Tables:    len(doc.Tables),
		Functions: len(doc.Functions),
	}
}
