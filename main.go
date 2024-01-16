package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	_ "github.com/bacalhau-project/bacalhau/pkg/version"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

//	@title			Bacalhau API
//	@version		v1
//	@description	The Bacalhau API is a RESTful API that allows you to interact with the Bacalhau network.
//	@termsOfService	http://bacalhau.org/terms/

// TODO: #3165 Host the terms of service on bacalhau.org/terms

//	@contact.name	API Support
//	@contact.url	https://www.expanso.io/contact/
//	@contact.email	support@bacalhau.org

// TODO: #3166 Host an email address and a contact form on bacalhau.org/support

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:1234
//	@BasePath	/api/v1

// @externalDocs.description 	Bacalhau Documentation
// @externalDocs.url			https://docs.bacalhau.org

////	@securityDefinitions.basic	BasicAuth

////	@securityDefinitions.apikey	ApiKeyAuth
////	@in							header
////	@name						Authorization
////	@description				Description for what is this security definition being used

// -- Add authentication to swagger here
//// 	@securitydefinitions.oauth2.application	OAuth2Application
//// 	@tokenUrl								https://example.com/oauth/token
//// 	@scope.write							Grants write access
//// 	@scope.admin							Grants read and write access to administrative information
//
//// 	@securitydefinitions.oauth2.implicit	OAuth2Implicit
//// 	@authorizationUrl						https://example.com/oauth/authorize
//// 	@scope.write							Grants write access
//// 	@scope.admin							Grants read and write access to administrative information
//
//// 	@securitydefinitions.oauth2.password	OAuth2Password
//// 	@tokenUrl								https://example.com/oauth/token
//// 	@scope.read								Grants read access
//// 	@scope.write							Grants write access
//// 	@scope.admin							Grants read and write access to administrative information
//
//// 	@securitydefinitions.oauth2.accessCode	OAuth2AccessCode
//// 	@tokenUrl								https://example.com/oauth/token
//// 	@authorizationUrl						https://example.com/oauth/authorize
//// 	@scope.admin							Grants read and write access to administrative information

func main() {
	defer func() {
		// Make sure any buffered logs are written if something failed before logging was configured.
		logger.LogBufferedLogs(nil)
	}()

	_ = godotenv.Load()

	devstackEnvFile := config.DevstackEnvFile()
	if _, err := os.Stat(devstackEnvFile); err == nil {
		log.Debug().Msgf("Loading environment from %s", devstackEnvFile)
		_ = godotenv.Overload(devstackEnvFile)
	}

	// Ensure commands are able to stop cleanly if someone presses ctrl+c
	ctx, cancel := signal.NotifyContext(context.Background(), util.ShutdownSignals...)
	defer cancel()

	cli.Execute(ctx)
}
