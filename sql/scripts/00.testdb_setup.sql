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
    FileName varchar(255),
    FileType varchar(9),
    FileID varchar(50),
    Extension varchar(25),
    ParentID varchar(255),
    FilePath varchar(5120),
    FileSize int,
    ModifiedOn DATETIME,
    DeletedOn DATETIME,
    PRIMARY KEY (FileID),
    CONSTRAINT FK_File_UserAccount
        FOREIGN KEY (AccountID) 
        REFERENCES UserAccount(AccountID)
        ON DELETE CASCADE
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

/* default entries for the test database */
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
        ".txt",
        NULL,
        "./testapp/89672a64-f3ff-490c-8f2d-7e5cf5d4aa70/randomfileidhere",
        1234,
        NOW(),
        NULL
    ),
    (
        "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70",
        "folder1",
        "dir",
        "randomfolderidhere",
        "",
        NULL,
        "",
        101,
        NOW(),
        NULL
    ),
    (
        "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70",
        "test2",
        "file",
        "anotherfileidhere",
        ".txt",
        "randomfolderidhere",
        "./testapp/89672a64-f3ff-490c-8f2d-7e5cf5d4aa70/anotherfileidhere",
        1234,
        NOW(),
        NULL
    );
INSERT INTO Session
    VALUES
    (
        "7ca90f85-b1e0-4214-8ff6-4e3720cc8078",
        "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70", 
        NOW(), 
        DATE_ADD(NOW(), INTERVAL 14 DAY)
    );