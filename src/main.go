//main
package main
//imports
import (
	"context"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"time"
	"encoding/json"
    "fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/joho/godotenv"
	"github.com/go-playground/validator"
	//"golang.org/x/crypto/acme/autocert"
)

//data schemas
type (
	user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	SMSMessage struct {
		Message string    `json:"message"`
		Number string     `json:"number" validate:"required"`
		Provider string   `json:"provider"`
	}
	
	ProviderInfo struct {
		SMS_ADDRESS string `json:"SMS_ADDRESS"`
		MMS_ADDRESS string `json:"MMS_ADDRESS"`
	}

	ResponseMessage struct {
		Message string `json:"message"`
		Success bool   `json:"success"`
	}

	CustomValidator struct {
		validator *validator.Validate
	}

	//holds all env file data
	Config struct {
		EMAIL string
		PASS string
		MAIL_SERVER string
		PORT string
		PROVIDERS_FILENAME string
		KNOWN_NUMBERS_FILENAME string
	}
)

var (
	users = map[int]*user{}
	seq   = 1
	knownNumbers = map[string]string(nil)
	knownProviders = map[string]ProviderInfo(nil)
	config = new(Config)
)

// custom validator for sms body
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

//----------
// Handlers
//----------
func handleSMS(c echo.Context) (err error) {
	log.Info("handling SMS")
	msg := new(SMSMessage)
	// map the data from the context to the msg
	if err = c.Bind(msg); err != nil {
		return
	}
	if err = c.Validate(msg); err != nil {
		return
	}
	log.Info(msg)

	providerPath := formProviderPath(msg)
	sendMessage(providerPath, msg.Message)

	res := new(ResponseMessage)
	res.Success = true
	res.Message = "message was sent to the number"
	//c.JSON(http.StatusOK, "ok")
	//return c.NoContent(http.StatusNoContent)
	return c.JSONPretty(http.StatusOK, res, "  ")
}

//----------
// Utils
//----------

func getDataPath() string {
	fp, _ := filepath.Abs("./app/data/")
	return fp
}

// return env var values if they exist else fallback
func getEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return fallback
}

func formProviderPath(msg *SMSMessage) string {
	smsEmail := ""
	if msg.Provider != "" {
		// provider is known
		prov := knownProviders[msg.Provider]
		smsEmail = msg.Number + prov.SMS_ADDRESS
	} else {
		// provider unknown
		prov := knownProviders[knownNumbers[msg.Number]]
		smsEmail = msg.Number + prov.SMS_ADDRESS
	}

	return smsEmail
} 

//methods
func populateProviders() map[string]ProviderInfo {
	// we initialize our provider data
	var providers map[string]ProviderInfo
	fp := path.Join( getDataPath(), config.PROVIDERS_FILENAME)
	log.Info("opening: " + fp)
	jsonFile, err := os.Open(fp)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Error(err)
		return providers
	}
	fmt.Println("Successfully Opened "+ config.PROVIDERS_FILENAME)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	// read our opened file as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we unmarshal our byteArray which contains our
	// jsonFile's content into data which we defined above
	json.Unmarshal(byteValue, &providers)
    fmt.Println(providers)
	return providers
}

func populateKnownNumbers() map[string]string {
	// load a list of know numbers and providers
	fp := path.Join( getDataPath(), config.KNOWN_NUMBERS_FILENAME)
	// Open our jsonFile
    jsonFile, err := os.Open(fp)
    // if we os.Open returns an error then handle it
    if err != nil {
        fmt.Println(err)
    }
    // defer the closing of our jsonFile so that we can parse it later on
    defer jsonFile.Close()

    byteValue, _ := ioutil.ReadAll(jsonFile)

    var result map[string]string
	json.Unmarshal([]byte(byteValue), &result)
	fmt.Println(result)
	return result
}

func sendMessage(emailAddress string, txt string) {
	log.Info("sending: " + txt + " to: " + emailAddress)
	// Set up authentication information.
	auth := smtp.PlainAuth("", config.EMAIL, config.PASS, config.MAIL_SERVER)
	

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	to := []string{emailAddress}
	msg := []byte("To: "+emailAddress+"\r\n" +
		"Subject: discount Gophers!\r\n" +
		"\r\n" +
		"This is the email body.\r\n")
	err := smtp.SendMail(config.MAIL_SERVER+":25", auth, config.EMAIL, to, msg)
	if err != nil {
		log.Fatal(err)
	}
}

func setup() {
	if _, ok := os.LookupEnv("EMAIL"); ok {
		log.Info("found env vars")
	}else{
		//load config from .env if not set
		log.Info("loading from .env file")
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}
	// set config from env
	config.EMAIL = getEnv("EMAIL", "derp@face.com")
	config.PASS = getEnv("PASS", "")
	config.MAIL_SERVER = getEnv("MAIL_SERVER", "")
	config.PORT = getEnv("PORT", "443")
	config.PROVIDERS_FILENAME = getEnv("PROVIDERS_FILENAME", "providers.json")
	config.KNOWN_NUMBERS_FILENAME = getEnv("KNOWN_NUMBERS_FILENAME", "knownNumbers.json")

	//loda data
	knownProviders = populateProviders()
	knownNumbers = populateKnownNumbers()
	log.Info("known providers: ", knownProviders)
	log.Info("known numbers: ", knownNumbers)
}


func main() {
	//setup any config
	setup()
	
	// instansiate echo server
	e := echo.New()
	// Debug mode
	e.Debug = true
	// e.AutoTLSManager.HostPolicy = autocert.HostWhitelist("<DOMAIN>")
	// Cache certificates
	// e.AutoTLSManager.Cache = autocert.DirCache("/var/www/.cache")
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{http.MethodGet, http.MethodPost},
		//AllowHeaders: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))

	e.Validator = &CustomValidator{validator: validator.New()}

	e.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, `
			<h1>Welcome to Echo!</h1>
			<h3>TLS certificates automatically installed from Let's Encrypt :)</h3>
		`)
	})

	// Routes
	e.POST("/sms", handleSMS) 

	e.Logger.SetLevel(log.INFO)

	// Start server
	go func() {
		if err := e.Start(":"+ config.PORT); err != nil {
			e.Logger.Info("shutting down the server")
		}
		/*
		if err := e.StartAutoTLS(":"+ config.PORT); err != nil {
			e.Logger.Info("shutting down the server")
		}
		*/
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds.
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

/*
AT&T: number@txt.att.net (SMS), number@mms.att.net (MMS)
T-Mobile: number@tmomail.net (SMS & MMS)
Verizon: number@vtext.com (SMS), number@vzwpix.com (MMS)
Sprint: number@messaging.sprintpcs.com (SMS), number@pm.sprint.com (MMS)
XFinity Mobile: number@vtext.com (SMS), number@mypixmessages.com (MMS)
Virgin Mobile: number@vmobl.com (SMS), number@vmpix.com (MMS)
Tracfone: number@mmst5.tracfone.com (MMS)
Metro PCS: number@mymetropcs.com (SMS & MMS)
Boost Mobile: number@sms.myboostmobile.com (SMS), number@myboostmobile.com (MMS)
Cricket: number@sms.cricketwireless.net (SMS), number@mms.cricketwireless.net (MMS)
Republic Wireless: number@text.republicwireless.com (SMS)
Google Fi (Project Fi): number@msg.fi.google.com (SMS & MMS)
U.S. Cellular: number@email.uscc.net (SMS), number@mms.uscc.net (MMS)
Ting: number@message.ting.com
Consumer Cellular: number@mailmymobile.net
C-Spire: number@cspire1.com
Page Plus: number@vtext.com
*/