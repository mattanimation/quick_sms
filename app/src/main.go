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
	"strings"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/acme/autocert"
)

//data schemas
type (
	user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	SMSMessage struct {
		Message string        `json:"message"`
		Number string         `json:"number" validate:"required"`
		ProviderName string   `json:"providerName"`
	}

	Outlets struct {
		SMS string `json:"SMS"`
		MMS string `json:"MMS"`
	}
	
	Provider struct {
		Name string `json:"name`
		Outlets Outlets `json:"outlets"`
	}

	ResponseMessage struct {
		Message string `json:"message"`
		Success bool `json:"success"`
	}

	Config struct {
		EMAIL string
		PASS string
		MAIL_SERVER string
	}
)

var (
	users = map[int]*user{}
	seq   = 1
	knownNumbers = map[string]string(nil)
	knownProviders = []Provider(nil)
	config = new(Config)
)


//----------
// Handlers
//----------
func handleSMS(c echo.Context) (err error) {
	fmt.Println("handling SMS")
	msg := new(SMSMessage)
	if err = c.Bind(msg); err != nil {
		return
	}
	if err = c.Validate(msg); err != nil {
		return
	}
	// lookup provider by number
	for i, p := range knownProviders {
		fmt.Println(i, p)
		prov := knownProviders[i]
		parts := []string{knownNumbers[msg.Number], prov.Outlets.SMS}
		smsEmail := strings.Join(parts,"")
		if smsEmail != "" {
			sendMessage(smsEmail, msg.Message)
		}
	}

	res := new(ResponseMessage)
	res.Success = true
	res.Message = "message was sent to the number"
	//c.JSON(http.StatusOK, "ok")
	//return c.NoContent(http.StatusNoContent)
	return c.JSONPretty(http.StatusOK, res, "  ")
}

//methods
func populateProviders() []Provider {
	// TOOD: load from .env or yaml file?
	jsonFilename := "providers.json"
	jsonFile, err := os.Open(jsonFilename)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened providers.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	// read our opened xmlFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we initialize our Users array
	var providers []Provider

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &providers)
    fmt.Println(providers)
	return providers
}

func populateKnownNumbers() map[string]string {
	// load a list of know numbers and providers
	jsonFilename := "knownNumbers.json"
	// Open our jsonFile
    jsonFile, err := os.Open(jsonFilename)
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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	config.EMAIL = os.Getenv("EMAIL")
	config.PASS = os.Getenv("PASS")
	config.MAIL_SERVER = os.Getenv("MAIL_SERVER")

}

func main() {
	//setup any config
	setup()
	
	e := echo.New()
	// Debug mode
	e.Debug = true
	// e.AutoTLSManager.HostPolicy = autocert.HostWhitelist("<DOMAIN>")
	// Cache certificates
	e.AutoTLSManager.Cache = autocert.DirCache("/var/www/.cache")
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))

	e.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, `
			<h1>Welcome to Echo!</h1>
			<h3>TLS certificates automatically installed from Let's Encrypt :)</h3>
		`)
	})

	knownProviders = populateProviders()
	knownNumbers = populateKnownNumbers()

	// Routes
	e.POST("/sms", handleSMS) 

	e.Logger.SetLevel(log.INFO)

	// Start server
	go func() {
		/*
		if err := e.Start(":9000"); err != nil {
			e.Logger.Info("shutting down the server")
		}
		*/
		if err := e.StartAutoTLS(":443"); err != nil {
			e.Logger.Info("shutting down the server")
		}
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