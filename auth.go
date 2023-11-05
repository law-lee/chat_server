package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/stretchr/gomniauth"
	gomniauthcommon "github.com/stretchr/gomniauth/common"
	"github.com/stretchr/objx"
)

//We have concluded that our GetAvatarURL method depends on a type that is not available
//to us at the point we need it, so what would be a good alternative? We could pass each
//required field as a separate argument, but this would make our interface brittle, since as
//soon as an Avatar implementation needs a new piece of information, we'd have to change
//the method signature. Instead, we will create a new type that will encapsulate the
//information our Avatar implementations need while conceptually remaining decoupled
//from our specific case.
type ChatUser interface {
	UniqueID() string
	AvatarURL() string
}

//makes use of a very interesting feature in Go: type embedding. We actually embedded the
//gomniauth/common.User interface type, which means that our struct interface
//implements the interface automatically.
type chatUser struct {
	gomniauthcommon.User
	uniqueID string
}

//UniqueID You may have noticed that we only actually implemented one of the two required methods
//to satisfy our ChatUser interface. We got away with this because the Gomniauth User
//interface happens to define the same AvatarURL method. In practice, when we instantiate
//our chatUser struct provided we set an appropriate value for the implied Gomniauth User
//field our object implements both Gomniauth's User interface and our own ChatUser
//interface at the same time.
func (u chatUser) UniqueID() string {
	return u.uniqueID
}

//OAuth2 is an open authorization standard designed to allow resource owners to give clients
//delegated access to private data (such as wall posts or tweets) via an access token exchange
//handshake. Even if you do not wish to access the private data, OAuth2 is a great option that
//allows people to sign in using their existing credentials, without exposing those credentials
//to a third-party site. In this case, we are the third party, and we want to allow our users to
//sign in using services that support OAuth2.
type authHandler struct {
	next http.Handler
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie("auth")
	if errors.Is(err, http.ErrNoCookie) {
		// not authenticated
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}
	if err != nil {
		// some other error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// success - call the next handler
	h.next.ServeHTTP(w, r)
}
func MustAuth(handler http.Handler) http.Handler {
	return &authHandler{next: handler}
}

// loginHandler handles the third-party login process.
// format: /auth/{action}/{provider}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	segs := strings.Split(r.URL.Path, "/")
	action := segs[2]
	provider := segs[3]
	switch action {
	case "login":
		provider, err := gomniauth.Provider(provider)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error when trying to get provider %s: %s", provider, err), http.StatusBadRequest)
			return
		}
		loginUrl, err := provider.GetBeginAuthURL(nil, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error when trying to GetBeginAuthURL for %s:%s", provider, err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Location", loginUrl)
		w.WriteHeader(http.StatusTemporaryRedirect)
	case "callback":
		provider, err := gomniauth.Provider(provider)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error when trying to get provider %s: %s",
				provider, err), http.StatusBadRequest)
			return
		}
		//parse RawQuery from the request into
		//objx.Map (the multipurpose map type that Gomniauth uses), and the CompleteAuth
		//method uses the values to complete the OAuth2 provider handshake with the provider. All
		//being well, we will be given some authorized credentials with which we will be able to
		//access our user's basic data
		creds, err := provider.CompleteAuth(objx.MustFromURLQuery(r.URL.RawQuery))
		if err != nil {
			http.Error(w, fmt.Sprintf("Error when trying to complete auth for %s: %s",
				provider, err), http.StatusInternalServerError)
			return
		}
		user, err := provider.GetUser(creds)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error when trying to get user from %s: %s",
				provider, err), http.StatusInternalServerError)
			return
		}
		//Base64-encoding data ensures it won't contain any special or
		//unpredictable characters, which is useful for situations such as passing
		//data to a URL or storing it in a cookie.
		//authCookieValue := objx.New(map[string]interface{}{
		//	"name": user.Name(),
		//	//make use of the avatar URL field
		//	"avatar_url": user.AvatarURL(),
		//	"email":      user.Email(),
		//}).MustBase64()
		chatUser := &chatUser{User: user}
		m := md5.New()
		io.WriteString(m, strings.ToLower(user.Email()))
		chatUser.uniqueID = fmt.Sprintf("%x", m.Sum(nil))
		avatarURL, err := avatars.GetAvatarURL(chatUser)
		if err != nil {
			log.Fatalln("Error when trying to GetAvatarURL", "-", err)
		}
		authCookieValue := objx.New(map[string]interface{}{
			"userid":     chatUser.uniqueID,
			"name":       user.Name(),
			"avatar_url": avatarURL,
			"email":      user.Email(),
		}).MustBase64()
		http.SetCookie(w, &http.Cookie{
			Name:  "auth",
			Value: authCookieValue,
			Path:  "/"})
		w.Header().Set("Location", "/chat")
		w.WriteHeader(http.StatusTemporaryRedirect)
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Auth action %s not supported", action)
	}
}
