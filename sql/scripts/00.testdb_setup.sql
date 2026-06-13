CREATE DATABASE IF NOT EXISTS TestLocalCloudStorage;
USE TestLocalCloudStorage;

CREATE TABLE IF NOT EXISTS UserAccount(
    AccountID varchar(255),
    Username varchar(30) NOT NULL UNIQUE,
    PasswordHash varchar(255) NOT NULL,
    CreatedOn DATETIME,
    Active BOOL DEFAULT 1,
    PRIMARY KEY (AccountID)
);

CREATE TABLE IF NOT EXISTS File(
    AccountID varchar(255) NOT NULL,
    FileName varchar(255) NOT NULL,
    FileType varchar(9) NOT NULL,
    FileID varchar(50) NOT NULL,
    Extension varchar(25) NOT NULL,
    ParentID varchar(255) NOT NULL,
    FilePath varchar(5120) NOT NULL,
    FileSize int NOT NULL,
    ModifiedOn DATETIME NOT NULL,
    DeletedOn DATETIME,
    UploadStatus varchar(10) NOT NULL,
    UniqueHash varchar(64) NOT NULL,
    PRIMARY KEY (FileID),
    CONSTRAINT FK_File_UserAccount
        FOREIGN KEY (AccountID) 
        REFERENCES UserAccount(AccountID)
        ON DELETE CASCADE
);

-- prevents duplicate entries for normal files who share
-- the same parent ID on the same account
-- folders can have duplicate entries
-- the unique hash is a sha256 of: account id + file name + file extension + parent id
CREATE UNIQUE INDEX FileUniqueIndex ON File(
    (CASE WHEN FileType = 'file' THEN UniqueHash END)
);

CREATE TABLE IF NOT EXISTS Session(
    SessionID varchar(255),
    AccountID varchar(255) NOT NULL UNIQUE,
    CreatedOn DATETIME,
    ExpireOn DATETIME,
    PRIMARY KEY (SessionID),
    CONSTRAINT FK_Session_UserAccount
        FOREIGN KEY (AccountID)
        REFERENCES UserAccount(AccountID)
        ON DELETE CASCADE
);

-- default entries for the test database, do not include below in prod
-- the path is not the full path and will be appended to the root folder
INSERT INTO UserAccount
    VALUES
    (
        "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70", 
        "test.username", 
        "$argon2id$v=19$m=65536,t=2,p=4$QTdpUkJ3c3J0amlOT2huV2VBR2duZw$vzICl8p5CVfpGfypDV4yIVULsYatAmir6B8nHWtcPtE", 
        NOW(), 
        1
    );
INSERT INTO File
    VALUES
    (
        "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70",
        "test1",
        "file",
        "randomfileidhere",
        "txt",
        "",
        "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70/randomfileidhere",
        0,
        NOW(),
        NULL,
        "completed",
        "f3bf6020372579f86aadbb42a6416de73463cef330a84c6a1cf493eea411a2dd"
    ),
    (
        "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70",
        "folder1",
        "dir",
        "randomfolderidhere",
        "",
        "",
        "",
        0,
        NOW(),
        NULL,
        "completed",
        ""
    ),
    (
        "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70",
        "test2",
        "file",
        "anotherfileidhere",
        "txt",
        "randomfolderidhere",
        "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70/anotherfileidhere",
        0,
        NOW(),
        NULL,
        "completed",
        "3b3cfd0153176b8c917c784adbcb6026b004f3514af0711b4c19f50829969f9d"
    );
INSERT INTO Session
    VALUES
    (
        "7ca90f85-b1e0-4214-8ff6-4e3720cc8078",
        "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70", 
        NOW(), 
        DATE_ADD(NOW(), INTERVAL 14 DAY)
    );