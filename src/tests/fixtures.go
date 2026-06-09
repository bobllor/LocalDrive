package tests

// DbConInfo is a read-only struct used to hold information
// of the test database.
var DbConInfo = TestDbCon{
	User:     "root",
	Addr:     ":3307",
	Password: "",
	Net:      "tcp",
	DbName:   "TestLocalCloudStorage",
}

// DbRowInfo is a read-only variable that contains the default values
// included in the test database.
var DbRowInfo = TestDbRow{
	AccountID:  "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70",
	Username:   "test.username",
	UserActive: true,
	PhcString:  "$argon2id$v=19$m=65536,t=2,p=4$QTdpUkJ3c3J0amlOT2huV2VBR2duZw$vzICl8p5CVfpGfypDV4yIVULsYatAmir6B8nHWtcPtE",
	SessionID:  "7ca90f85-b1e0-4214-8ff6-4e3720cc8078",
	FileID:     "randomfileidhere",
	FileName:   "test1",
	UniqueHash: "f3bf6020372579f86aadbb42a6416de73463cef330a84c6a1cf493eea411a2dd",
}

// TestPassword is the test password used to create the PhcString for
// the default entry in the test database.
var TestPassword = "anothertestpassword"

// TestSalt is the salt used to salt the test password for the
// default entry in the test database.
var TestSalt = []byte("A7iRBwsrtjiNOhnWeAGgng")
