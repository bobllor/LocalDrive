package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/bobllor/cloud-project/src/api"
	"github.com/bobllor/cloud-project/src/config"
	dbgateway "github.com/bobllor/cloud-project/src/db_gateway"
	"github.com/bobllor/cloud-project/src/server"
	"github.com/bobllor/cloud-project/src/utils"
	"github.com/bobllor/gologger"
)

func main() {
	fileName := "config"
	exts := []string{".yaml", ".yml"}

	configFile, err := getConfig(fileName, exts)
	if err != nil {
		log.Fatal(err)
	}

	scfg, err := config.NewServerConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: add this to the config file
	// TODO: add real logging for prod
	logger := gologger.NewLogger(log.New(os.Stdout, "", log.Ltime|log.Ldate), gologger.Lsilent)

	err = scfg.LoadEnv()
	if err != nil {
		log.Fatal(err)
	}

	filePw := os.Getenv(config.EnvFilePwKey)
	userPw := os.Getenv(config.EnvUserPwKey)

	dbName := scfg.Database.Name
	network := scfg.Database.NetProtocol
	dbAddr := scfg.Database.Address

	fileConfig := dbgateway.NewConfig(
		scfg.Database.FileUser.User,
		filePw,
		network,
		dbAddr,
		dbName,
	)
	userConfig := dbgateway.NewConfig(
		scfg.Database.AccountUser.User,
		userPw,
		network,
		dbAddr,
		dbName,
	)

	fdb, err := dbgateway.NewDatabase(fileConfig)
	if err != nil {
		log.Fatal(err)
	}
	udb, err := dbgateway.NewDatabase(userConfig)
	if err != nil {
		log.Fatal(err)
	}

	deps := utils.NewDeps(logger)

	fg := dbgateway.NewFileGateway(fdb, deps)
	ug := dbgateway.NewUserGateway(udb, deps)
	sg := dbgateway.NewSessionGateway(udb, deps)
	ds := dbgateway.DirectoryStore{
		Storage: scfg.StoragePath,
		Temp:    os.TempDir(),
	}

	gw := dbgateway.NewGateway(fg, ug, sg, ds)

	serv, err := createServer(gw, logger, scfg.ServerAddress)
	if err != nil {
		log.Fatal(err)
	}

	logger.Info("Starting server")
	log.Fatal(serv.Start())
}

// getConfig retrieves the config file from the root folder.
//
// It loops through a given extension slice for the file with the config
// name. If the name is found with the
// extension, then it will return the full path to the file.
//
// The given extensions must start with a leading period. It will return an error
// if the extensions do not follow this convention.
//
// If no files are found or an error occurs, then it will
// return an error instead.
func getConfig(configName string, exts []string) (string, error) {
	if len(exts) == 0 {
		return "", errors.New("no extensions given for config search")
	}

	const EXT_PATTERN = `^\.(.+)$`

	rootPath, err := os.Getwd()
	if err != nil {
		return "", err
	}

	files, err := utils.GetFiles(rootPath)
	if err != nil {
		return "", err
	}

	for _, ext := range exts {
		if len(ext) == 0 {
			return "", errors.New("empty extension string given in extensions")
		}

		matched, err := regexp.MatchString(EXT_PATTERN, ext)
		if err != nil {
			return "", err
		}
		if !matched {
			return "", fmt.Errorf("'%s' must contain a leading period", ext)
		}

		fileName := strings.ToLower(fmt.Sprintf("%s%s", configName, ext))

		filePath, ok := files[fileName]
		if ok {
			return filePath, nil
		}
	}

	return "", fmt.Errorf("could not find file %s", configName)
}

// createServer creates the server and registers the routes. It will
// return the server.Server.
func createServer(gw *dbgateway.Gateway, logger *gologger.Logger, serverAddress string) (*server.Server, error) {
	ap := api.NewApiHandler(gw, logger)
	serv, err := server.NewServer(serverAddress)
	if err != nil {
		return nil, err
	}

	serv.RegisterHandler(api.SessionGetValidateSessionRoute, ap.CreateRequestMiddleware(ap.SessionHandler.GetValidateSession))

	/* user handlers */
	serv.RegisterHandler(api.UserPostRegisterRoute, ap.CreateRequestMiddleware(ap.UserHandler.PostRegisterUser))
	serv.RegisterHandler(api.UserPostLoginRoute, ap.CreateRequestMiddleware(ap.UserHandler.PostLogin))
	serv.RegisterHandler(api.UserPostLogoutRoute, ap.CreateAuthMiddleware(ap.UserHandler.PostLogout))
	serv.RegisterHandler(api.UserGetUserRoute, ap.CreateAuthMiddleware(ap.UserHandler.GetUserBySessionID))

	/* file handlers */
	// handles both dynamic and root based access
	serv.RegisterHandler(api.FileGetFileRootRoute, ap.CreateAuthMiddleware(ap.FileHandler.GetFiles))
	serv.RegisterHandler(api.FileGetFileParentRoute, ap.CreateAuthMiddleware(ap.FileHandler.GetFiles))
	serv.RegisterHandler(api.FileGetDeletedFilesRoute, ap.CreateAuthMiddleware(ap.FileHandler.GetDeletedFiles))

	serv.RegisterHandler(api.FilePostDownloadFileRoute, ap.CreateAuthMiddleware(ap.FileHandler.DownloadFile))
	serv.RegisterHandler(api.FilePostAddFolderRoute, ap.CreateAuthMiddleware(ap.FileHandler.PostAddFolder))

	// file modifications
	serv.RegisterHandler(api.FilePostUploadFileChunkRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadFileChunk))
	serv.RegisterHandler(api.FilePostUploadFileCompleteRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadFileComplete))
	serv.RegisterHandler(api.FilePostUploadFileRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadGenerateId))

	serv.RegisterHandler(api.FilePatchUpdateFileStatusRoute, ap.CreateRequestMiddleware(ap.FileHandler.UploadFileStatusFailed))
	serv.RegisterHandler(api.FilePatchRenameFileRoute, ap.CreateAuthMiddleware(ap.FileHandler.RenameFile))
	serv.RegisterHandler(api.FileDeleteFileDeleteionRoute, ap.CreateAuthMiddleware(ap.FileHandler.DeleteFile))

	// other
	serv.RegisterHandler(api.FileGetFolderBreadcrumbsRoute, ap.CreateAuthMiddleware(ap.FileHandler.GetFolderBreadcrumbs))

	return serv, nil
}
