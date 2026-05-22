package tests

type TestDbCon struct {
	User     string
	Addr     string
	Password string
	Net      string
	DbName   string
}

type TestDbRow struct {
	AccountID  string
	Username   string
	UserActive bool
	PhcString  string
	SessionID  string
	FileID     string
	FileName   string
}
