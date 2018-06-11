package webircgateway

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/thebeerbarian/webircgateway/pkg/ircservices"
)

// Struct ircservices contains all fields for
var (
	 Atheme       *atheme.Atheme
	 nsCookieName     = "IRCSERVICEAUTH"
	 nsCookieHashKey  = []byte("MY_IRCSERVICEAUTH_HASH_KEY")
	 DEBUG	=     1
	 INFO	=     2
	 WARN	=     3
)

func ircservicesHTTPHandler(router *http.ServeMux) {
	router.HandleFunc("/webirc/ircservices/", func(w http.ResponseWriter, r *http.Request) {
		
		switch r.Method {
		case "GET":
		       logOut(DEBUG, "Request method: %s", r.Method)
		       ircservicesRespond(w, r)
		case "POST":
		       logOut(DEBUG, "Request method: %s", r.Method)
		       ircservicesCommand(w, r)
		default:
		       logOut(DEBUG, "Invalid request method: %s", r.Method)
		       return
		}
	})
}

func ircservicesCommand(w http.ResponseWriter, r *http.Request) {
	var (
	        err error
		authcookie = "*"
		account    = ""
		ipaddr     = r.Header.Get("X-Forwarded-For") //TODO: may not be proxied. Need fallback.
	        s          = securecookie.New(nsCookieHashKey, nil)
	)

        if err := r.ParseForm(); err != nil {
	        http.Error(w, "Error reading POST form data", http.StatusInternalServerError)
		return
	}

        // Check for Secure Cookie.
        if cookie, err := r.Cookie(nsCookieName); err == nil {
                value := make(map[string]string)
                // try to decode it
                if err = s.Decode(nsCookieName, cookie.Value, &value); err == nil {
			authcookie = value["authcookie"]
			account    = value["account"]
			ipaddr     = value["ipaddr"]
			logOut(DEBUG, "Retreived nsCookieName. authcookie: '%s' account: '%s' ipaddr: '%s'", value["authcookie"], value["account"], value["ipaddr"])
                }
        }
	
	// No valid authcookie, login required from form data.
        if authcookie == "*" {
	
	        nick := r.PostFormValue("nick")
	        password := r.PostFormValue("password")

	        //TODO: add services url to config.conf.
                Atheme, err = atheme.NewAtheme("http://127.0.0.1:8080/xmlrpc")
	
	        if err != nil {
	                logOut(WARN, "%s", err)
			return
	        }
	
	        if Atheme == nil {
	                logOut(WARN, "Atheme is nil")
			return
	        }
	
	        err = Atheme.Login(nick, password)
	
	        if err != nil {
	                logOut(WARN, "Atheme error: %s", err.Error())
			return
	        }
		
	        // Valid auth.  Generate and store encoded cookie.
	        if Atheme.Authcookie != "*" {
			authcookie = Atheme.Authcookie
			account    = Atheme.Account
	                value := map[string]string{
		                "authcookie": Atheme.Authcookie,
		                "account": Atheme.Account,
		                "ipaddr": ipaddr,
		        }

                        if encoded, err := s.Encode(nsCookieName, value); err == nil {
		                cookie := &http.Cookie{
			                Name:    nsCookieName,
				        Value:   encoded,
				        Domain:  ".beerbarian.com", //TODO: Needs config.conf setting.
			        }
			        logOut(DEBUG, "cookie ", cookie)
			        http.SetCookie(w, cookie)
				logOut(DEBUG, "Stored nsCookieName. authcookie: '%s', account: '%s' ipaddr: '%s'", Atheme.Authcookie, Atheme.Account, ipaddr)
		        }
	        }
	}
	
	
	out, _ := json.Marshal(map[string]interface{}{
		"authcookie":	authcookie,
		"account":	account,
		"ipaddr":	ipaddr,
	})
	

	w.Write(out)
}

// Generate a temporary developer page to post data to verify services functions are working.
// Checks will be added to check if user has valid authcookie and give status information
//   for their registration otherwise prompt for login.  Maybe have a config to enable/disable.

func ircservicesRespond(w http.ResponseWriter, r *http.Request) {

        var testpage string

        testpage = "<!DOCTYPE html>\n"
        testpage += "<html>\n"
        testpage += "  <head>\n"
        testpage += "    <title>IRC Services Test</title>\n"
        testpage += "  </head>\n"
        testpage += "  <div style=\"margin: 0 auto;padding: 0;width: 800px;\"><body>\n"
        testpage += "    <h1 style=\"color: black; font-family: verdana; text-align: center;\">IRC Services Test</h1>\n"
        testpage += "    <form action=\"/webirc/ircservices/\" method=\"post\">\n"
        testpage += "    <div style=\"text-align: center;margin: 0 auto;\">\n"
        testpage += "      <label for=\"nick\">Nickname</label> <input type=\"text\" name=\"nick\">&nbsp;&nbsp;\n"
        testpage += "      <label for=\"password\">Password</label> <input type=\"password\" name=\"password\"><br>\n"
        testpage += "      <button type=\"Submit\" value=\"Submit\">Submit</button>\n"
        testpage += "    </div></form>\n"
        testpage += "  </body></div>\n"
        testpage += "</html>\n"
       fmt.Fprint(w, testpage)

}

