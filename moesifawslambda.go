package moesifawslambda

import (
	"github.com/aws/aws-lambda-go/events"
	models "github.com/moesif/moesifapi-go/models"
	moesifapi "github.com/moesif/moesifapi-go"
	"log"
	"context"
	"os"
)

 // Global variable
 var (
	apiClient moesifapi.API
	debug bool
	logBody bool
	moesifOption map[string]interface{}
)

// Initialize the client
func moesifClient(moesifOption map[string]interface{}) {

	applicationId := os.Getenv("MOESIF_APPLICATION_ID")
	api := moesifapi.NewAPI(applicationId)
	apiClient = api

	//  Disable debug by default
	debug = false
	// Try to fetch the debug from the option
	if isDebug, found := moesifOption["Debug"].(bool); found {
			debug = isDebug
	}

	// Enable logBody by default
	logBody = true
	
	// Try to fetch the logBody from the option
	if isEnabled, found := moesifOption["Log_Body"].(bool); found {
		logBody = isEnabled
	}
}

func sendMoesifAsync(request events.APIGatewayProxyRequest, response events.APIGatewayProxyResponse, configurationOption map[string]interface{}) {

	// Api Version
	var apiVersion *string = nil
	 if isApiVersion, found := moesifOption["Api_Version"].(string); found {
		 apiVersion = &isApiVersion
	 }

	// Get Metadata
	var metadata map[string]interface{} = nil
	if _, found := moesifOption["Get_Metadata"]; found {
		metadata = moesifOption["Get_Metadata"].(func(events.APIGatewayProxyRequest, events.APIGatewayProxyResponse) map[string]interface{})(request, response)
	}

	// Get User
	var userId string
	if _, found := moesifOption["Identify_User"]; found {
		userId = moesifOption["Identify_User"].(func(events.APIGatewayProxyRequest, events.APIGatewayProxyResponse) string)(request, response)
	}

	// Get Company
	var companyId string
	if _, found := moesifOption["Identify_Company"]; found {
		companyId = moesifOption["Identify_Company"].(func(events.APIGatewayProxyRequest, events.APIGatewayProxyResponse) string)(request, response)
	}

	// Get Session Token
	var sessionToken string
	if _, found := moesifOption["Get_Session_Token"]; found {
		sessionToken = moesifOption["Get_Session_Token"].(func(events.APIGatewayProxyRequest, events.APIGatewayProxyResponse) string)(request, response)
	}

	// Prepare Moesif Event
	moesifEvent := prepareEvent(request, response, apiVersion, userId, companyId, sessionToken, metadata)

	// Should skip
	shouldSkip := false
	if _, found := moesifOption["Should_Skip"]; found {
		shouldSkip = moesifOption["Should_Skip"].(func(events.APIGatewayProxyRequest, events.APIGatewayProxyResponse) bool)(request, response)
	}

	if shouldSkip {
		if debug{
			log.Printf("Skip sending the event to Moesif")
		}
	} else {
		if debug {
			log.Printf("Sending the event to Moesif")
		}

		if _, found := moesifOption["Mask_Event_Model"]; found {
			moesifEvent = moesifOption["Mask_Event_Model"].(func(models.EventModel) models.EventModel)(moesifEvent)
		}

		// Call the function to send event to Moesif
		_, err := apiClient.CreateEvent(&moesifEvent)

		if err != nil {
			log.Fatalf("Error while sending event to Moesif: %s.\n", err.Error())
		}

		if debug {
			log.Printf("Successfully sent event to Moesif")
		}
	}
}

func MoesifLogger(f func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error), configurationOption map[string]interface{}) func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		response, err := f(ctx, request)
		// Call the function to initialize the moesif client and moesif options
		if apiClient == nil {
			moesifOption = configurationOption
			moesifClient(moesifOption)
		}
		sendMoesifAsync(request, response, configurationOption)
		return response, err
	}
}
