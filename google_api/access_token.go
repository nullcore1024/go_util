package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

type VoidedPurchaseRes struct {
	TokenPagination struct {
		NextPageToken string `json:"nextPageToken"`
	} `json:"tokenPagination"`
	VoidedPurchases []struct {
		Kind               string `json:"kind"`
		PurchaseToken      string `json:"purchaseToken"`
		PurchaseTimeMillis string `json:"purchaseTimeMillis"`
		VoidedTimeMillis   string `json:"voidedTimeMillis"`
		OrderID            string `json:"orderId"`
		VoidedSource       int    `json:"voidedSource"`
		VoidedReason       int    `json:"voidedReason"`
	} `json:"voidedPurchases"`
}

type ServerAccountKey struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

func serviceAccount(credentialFile string) (*oauth2.Token, error) {
	b, err := ioutil.ReadFile(credentialFile)
	if err != nil {
		return nil, err
	}
	c := ServerAccountKey{}
	json.Unmarshal(b, &c)
	config := &jwt.Config{
		Email:      c.Email,
		PrivateKey: []byte(c.PrivateKey),
		Scopes: []string{
			"https://www.googleapis.com/auth/androidpublisher",
		},
		TokenURL: google.JWTTokenURL,
	}
	token, err := config.TokenSource(oauth2.NoContext).Token()
	if err != nil {
		return nil, err
	}
	return token, nil
}

func main() {
	token, err := serviceAccount("credentials.json") // Please set here
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//wget https://www.googleapis.com/androidpublisher/v3/applications/com.tongitsgo.play/purchases/voidedpurchases?access_token=ya29.c.Ko8BwQfTkAZHewBkvC_n31fPsZZE_mdF9WHlqNp5bwRJ_onSCAtRKxvUTPgd5Rb44BR9PE7e81bl6d06h2VZSRQGUZUO7GDaMVL823h4XvjfV9DW3_XXWx9K7Ucft7By4th_MMsFGfVOW9VfYPvGBdSf4udMyf81nDDXXIGJ3Xier-3iKtB12sIUcQJv4Skpp-0
	fmt.Println(token)
}
